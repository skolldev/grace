from dataclasses import dataclass
from datetime import datetime, timezone
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlmodel import Session, select, text, bindparam

from server.core.auth import verify_api_key
from server.core.database import get_session
from server.models.models import (
    Device,
    DeviceSensor,
    Metric,
    SensorMetric,
    utc_now,
    to_naive_utc,
)
from server.models.schemas import (
    AggregatedMetricData,
    AggregatedMetricDataPoint,
    AggregatedNetworkData,
    AggregatedSensorDataPoint,
    AggregatedSensorMetricsResponse,
    AggregatedValue,
    MetricsPushPayload,
    MetricsResponse,
)
from server.core.logger import log_info

# Resolution bucket sizes in seconds
RESOLUTION_SECONDS = {
    "1m": 60,
    "5m": 300,
    "15m": 900,
    "1h": 3600,
    "6h": 21600,
    "1d": 86400,
}
VALID_RESOLUTIONS = set(RESOLUTION_SECONDS.keys())

# Maximum time range per resolution (in seconds) to prevent excessive queries
MAX_RANGE_SECONDS = {
    "1m": 86400,  # 1 day
    "5m": 86400 * 3,  # 3 days
    "15m": 86400 * 7,  # 1 week
    "1h": 86400 * 30,  # 1 month
    "6h": 86400 * 90,  # 3 months
    "1d": 86400 * 365,  # 1 year
}

router = APIRouter()


@dataclass
class ValidatedMetricsParams:
    """Validated parameters for metrics queries."""

    device: Device
    start: datetime
    end: datetime
    resolution: str
    session: Session


def validate_metrics_params(
    device_id: str,
    start: datetime = Query(..., description="Start time (ISO format)"),
    end: datetime = Query(..., description="End time (ISO format)"),
    resolution: str = Query(
        ..., description="Aggregation bucket: 1m, 5m, 15m, 1h, 6h, 1d"
    ),
    session: Session = Depends(get_session),
) -> ValidatedMetricsParams:
    """Dependency that validates common metrics query parameters."""
    # Verify device exists
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Validate resolution
    if resolution not in VALID_RESOLUTIONS:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid resolution. Valid values: {', '.join(sorted(VALID_RESOLUTIONS))}",
        )

    # Normalize to naive UTC
    start = to_naive_utc(start)
    end = to_naive_utc(end)

    # Validate time range
    if end <= start:
        raise HTTPException(
            status_code=400,
            detail="End time must be after start time",
        )

    # Check maximum time range for resolution
    max_range = MAX_RANGE_SECONDS[resolution]
    range_seconds = (end - start).total_seconds()
    if range_seconds > max_range:
        max_range_desc = _format_duration(max_range)
        raise HTTPException(
            status_code=400,
            detail=f"Time range too large for {resolution} resolution. Maximum: {max_range_desc}",
        )

    return ValidatedMetricsParams(device, start, end, resolution, session)


def _format_duration(seconds: int) -> str:
    """Format duration in seconds to human-readable string."""
    if seconds >= 86400:
        days = seconds // 86400
        return f"{days} day{'s' if days > 1 else ''}"
    elif seconds >= 3600:
        hours = seconds // 3600
        return f"{hours} hour{'s' if hours > 1 else ''}"
    else:
        minutes = seconds // 60
        return f"{minutes} minute{'s' if minutes > 1 else ''}"


def get_aggregated_metrics(
    session: Session,
    device_id: str,
    start: datetime,
    end: datetime,
    resolution: str,
) -> list[AggregatedMetricDataPoint]:
    """Fetch metrics and aggregate into time buckets."""
    bucket_seconds = RESOLUTION_SECONDS[resolution]

    # Format timestamps as ISO strings for consistent comparison
    # (avoids timezone issues with Unix timestamp conversion)
    start_str = start.strftime("%Y-%m-%d %H:%M:%S")
    end_str = end.strftime("%Y-%m-%d %H:%M:%S")

    # Do aggregation in SQL - direct column access (no JSON parsing)
    query = text(
        """
        SELECT
            (unixepoch(timestamp) / :bucket) * :bucket as bucket_ts,
            AVG(cpu_percent) as cpu_avg,
            MIN(cpu_percent) as cpu_min,
            MAX(cpu_percent) as cpu_max,
            AVG(ram_percent) as ram_avg,
            MIN(ram_percent) as ram_min,
            MAX(ram_percent) as ram_max,
            AVG(net_rx_bytes_sec) as net_rx_avg,
            MIN(net_rx_bytes_sec) as net_rx_min,
            MAX(net_rx_bytes_sec) as net_rx_max,
            AVG(net_tx_bytes_sec) as net_tx_avg,
            MIN(net_tx_bytes_sec) as net_tx_min,
            MAX(net_tx_bytes_sec) as net_tx_max
        FROM metrics
        WHERE device_id = :device_id
          AND timestamp >= :start_ts
          AND timestamp < :end_ts
        GROUP BY bucket_ts
        ORDER BY bucket_ts
    """
    )

    result = session.execute(
        query,
        {
            "bucket": bucket_seconds,
            "device_id": device_id,
            "start_ts": start_str,
            "end_ts": end_str,
        },
    )
    rows = result.fetchall()

    # Build lookup dict from SQL results
    data_by_ts: dict[int, AggregatedMetricDataPoint] = {}
    for row in rows:
        data_by_ts[row.bucket_ts] = AggregatedMetricDataPoint(
            timestamp=_format_ts(row.bucket_ts),
            data=AggregatedMetricData(
                cpu={"percent": _agg(row.cpu_avg, row.cpu_min, row.cpu_max)},
                ram={"percent": _agg(row.ram_avg, row.ram_min, row.ram_max)},
                network=AggregatedNetworkData(
                    rx_sec=_agg(row.net_rx_avg, row.net_rx_min, row.net_rx_max),
                    tx_sec=_agg(row.net_tx_avg, row.net_tx_min, row.net_tx_max),
                ),
            ),
        )

    # Generate all expected bucket timestamps and fill gaps with nulls
    expected_timestamps = _generate_bucket_timestamps(start, end, resolution)
    return [data_by_ts.get(ts) or _null_bucket(ts) for ts in expected_timestamps]


def _format_ts(ts: int) -> str:
    """Format Unix timestamp as ISO string with Z suffix."""
    return (
        datetime.fromtimestamp(ts, tz=timezone.utc).isoformat().replace("+00:00", "Z")
    )


def _agg(avg: float | None, min_: float | None, max_: float | None) -> AggregatedValue:
    """Return aggregated values with consistent structure (nulls for missing data)."""
    if avg is None:
        return AggregatedValue(avg=None, min=None, max=None)
    return AggregatedValue(
        avg=round(avg, 2),
        min=round(min_, 2),
        max=round(max_, 2),
    )


def _generate_bucket_timestamps(
    start: datetime, end: datetime, resolution: str
) -> list[int]:
    """Generate all expected bucket timestamps between start and end.

    Note: start and end are naive datetimes representing UTC.
    """
    bucket_seconds = RESOLUTION_SECONDS[resolution]

    # Convert naive UTC datetimes to Unix timestamps
    # (replace with UTC timezone to get correct timestamp)
    start_ts = int(start.replace(tzinfo=timezone.utc).timestamp())
    end_ts = int(end.replace(tzinfo=timezone.utc).timestamp())

    # Align start to bucket boundary
    aligned_start = (start_ts // bucket_seconds) * bucket_seconds

    timestamps = []
    current = aligned_start
    while current < end_ts:
        timestamps.append(current)
        current += bucket_seconds

    return timestamps


def _null_bucket(timestamp: int) -> AggregatedMetricDataPoint:
    """Create a bucket with null values for all metrics."""
    null_agg = _agg(None, None, None)
    return AggregatedMetricDataPoint(
        timestamp=_format_ts(timestamp),
        data=AggregatedMetricData(
            cpu={"percent": null_agg},
            ram={"percent": null_agg},
            network=AggregatedNetworkData(rx_sec=null_agg, tx_sec=null_agg),
        ),
    )


@router.post("/{device_id}/metrics", response_model=MetricsResponse)
def push_metrics(
    device_id: str,
    payload: MetricsPushPayload,
    session: Session = Depends(get_session),
    _: None = Depends(verify_api_key),
):
    # Verify device exists
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Update last seen
    device.last_seen_at = utc_now()
    session.add(device)

    timestamp = to_naive_utc(payload.timestamp or utc_now())

    # Extract metrics from payload and store as columns
    m = payload.metrics
    metric = Metric(
        device_id=device_id,
        timestamp=timestamp,
        cpu_percent=m.get("cpu", {}).get("percent"),
        ram_percent=m.get("ram", {}).get("percent"),
        ram_used_gb=m.get("ram", {}).get("used_gb"),
        ram_total_gb=m.get("ram", {}).get("total_gb"),
        disk=m.get("disk"),  # Store array as-is
        net_rx_bytes_sec=m.get("network", {}).get("rx_bytes_per_sec"),
        net_tx_bytes_sec=m.get("network", {}).get("tx_bytes_per_sec"),
    )

    metric_data = {
        "cpu_percent": metric.cpu_percent,
        "ram_percent": metric.ram_percent,
        "ram_used_gb": metric.ram_used_gb,
        "ram_total_gb": metric.ram_total_gb,
        "disk": metric.disk,
        "net_rx_bytes_sec": metric.net_rx_bytes_sec,
        "net_tx_bytes_sec": metric.net_tx_bytes_sec,
    }
    log_info(f"Pushed metrics for device {device_id}: {metric_data}", "metrics")
    session.add(metric)

    # Store sensor metrics (only enabled sensors)
    sensors_data = m.get("sensors", {})
    if sensors_data:
        enabled_sensors = session.exec(
            select(DeviceSensor.sensor_id)
            .where(DeviceSensor.device_id == device_id)
            .where(DeviceSensor.enabled)
        ).all()
        enabled_set = set(enabled_sensors)

        for sensor_id, value in sensors_data.items():
            if sensor_id in enabled_set and value is not None:
                sensor_metric = SensorMetric(
                    device_id=device_id,
                    sensor_id=sensor_id,
                    timestamp=timestamp,
                    value=value,
                )
                session.add(sensor_metric)

    session.commit()

    return MetricsResponse(status="ok")


@router.get("/{device_id}/metrics", response_model=list[AggregatedMetricDataPoint])
def get_metrics(
    params: ValidatedMetricsParams = Depends(validate_metrics_params),
) -> list[AggregatedMetricDataPoint]:
    """Get aggregated metrics for a device."""
    return get_aggregated_metrics(
        params.session, params.device.id, params.start, params.end, params.resolution
    )


def get_aggregated_sensor_metrics(
    session: Session,
    device_id: str,
    start: datetime,
    end: datetime,
    resolution: str,
) -> AggregatedSensorMetricsResponse:
    """Fetch sensor metrics and aggregate into time buckets, grouped by sensor."""
    bucket_seconds = RESOLUTION_SECONDS[resolution]

    sensors_query = text(
        """
        SELECT sensor_id, COALESCE(display_name, name) as name, sensor_type, COALESCE(unit, '') as unit
        FROM device_sensors
        WHERE device_id = :device_id AND enabled = 1
    """
    )

    sensor_map = {
        row.sensor_id: {
            "sensor_id": row.sensor_id,
            "name": row.name,
            "sensor_type": row.sensor_type,
            "unit": row.unit,
            "data_points": {},
        }
        for row in session.execute(sensors_query, {"device_id": device_id}).fetchall()
    }

    if not sensor_map:
        return AggregatedSensorMetricsResponse(sensors=[])

    enabled_sensor_ids = list(sensor_map.keys())

    metrics_query = text(
        """
        SELECT 
            sensor_id,
            (unixepoch(timestamp) / :bucket) * :bucket as bucket_ts,
            AVG(value) as avg,
            MIN(value) as min,
            MAX(value) as max
        FROM sensor_metrics
        WHERE device_id = :device_id
          AND sensor_id IN :sensor_ids
          AND timestamp >= :start
          AND timestamp < :end
        GROUP BY sensor_id, bucket_ts
    """
    )

    metrics_query = metrics_query.bindparams(bindparam("sensor_ids", expanding=True))

    metrics_rows = session.execute(
        metrics_query,
        {
            "bucket": bucket_seconds,
            "device_id": device_id,
            "start": start,
            "end": end,
            "sensor_ids": enabled_sensor_ids,
        },
    ).fetchall()

    for row in metrics_rows:
        if row.sensor_id in sensor_map:
            sensor_map[row.sensor_id]["data_points"][row.bucket_ts] = (
                AggregatedValue(
                    avg=round(row.avg, 2) if row.avg is not None else None,
                    min=round(row.min, 2) if row.min is not None else None,
                    max=round(row.max, 2) if row.max is not None else None,
                )
            )

    expected_ts = _generate_bucket_timestamps(start, end, resolution)
    null_val = AggregatedValue(avg=None, min=None, max=None)

    formatted_sensors = []

    for sensor in sensor_map.values():
        series = [
            AggregatedSensorDataPoint(
                timestamp=_format_ts(ts),
                value=sensor["data_points"].get(ts, null_val),
            )
            for ts in expected_ts
        ]

        formatted_sensors.append(
            {
                "sensor_id": sensor["sensor_id"],
                "name": sensor["name"],
                "sensor_type": sensor["sensor_type"],
                "unit": sensor["unit"],
                "data": series,
            }
        )

    return AggregatedSensorMetricsResponse(sensors=formatted_sensors)


@router.get(
    "/{device_id}/metrics/sensors", response_model=AggregatedSensorMetricsResponse
)
def get_sensor_metrics(
    params: ValidatedMetricsParams = Depends(validate_metrics_params),
) -> AggregatedSensorMetricsResponse:
    """Get aggregated sensor metrics for a device."""
    return get_aggregated_sensor_metrics(
        params.session, params.device.id, params.start, params.end, params.resolution
    )
