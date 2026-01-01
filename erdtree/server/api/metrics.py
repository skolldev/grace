from datetime import datetime, timezone
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlmodel import Session, text

from server.core.auth import verify_api_key
from server.core.database import get_session
from server.models.models import Device, Metric, utc_now
from server.models.schemas import MetricsPushPayload, MetricsResponse

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

router = APIRouter()


def aggregate(values: list[float]) -> dict | None:
    """Compute avg/min/max for a list of values."""
    if not values:
        return None
    return {
        "avg": round(sum(values) / len(values), 2),
        "min": round(min(values), 2),
        "max": round(max(values), 2),
    }


def get_aggregated_metrics(
    session: Session,
    device_id: str,
    start: datetime,
    end: datetime,
    resolution: str,
) -> list[dict]:
    """Fetch metrics and aggregate into time buckets."""
    bucket_seconds = RESOLUTION_SECONDS[resolution]

    # Convert to unix timestamps for direct comparison
    start_ts = int(start.timestamp())
    end_ts = int(end.timestamp())

    # Do aggregation in SQL - returns only ~100-300 rows instead of 60k+
    query = text(
        """
        SELECT
            (unixepoch(timestamp) / :bucket) * :bucket as bucket_ts,
            AVG(json_extract(data, '$.cpu.percent')) as cpu_avg,
            MIN(json_extract(data, '$.cpu.percent')) as cpu_min,
            MAX(json_extract(data, '$.cpu.percent')) as cpu_max,
            AVG(json_extract(data, '$.ram.percent')) as ram_avg,
            MIN(json_extract(data, '$.ram.percent')) as ram_min,
            MAX(json_extract(data, '$.ram.percent')) as ram_max,
            AVG(json_extract(data, '$.network.rx_bytes_per_sec')) as net_rx_avg,
            MIN(json_extract(data, '$.network.rx_bytes_per_sec')) as net_rx_min,
            MAX(json_extract(data, '$.network.rx_bytes_per_sec')) as net_rx_max,
            AVG(json_extract(data, '$.network.tx_bytes_per_sec')) as net_tx_avg,
            MIN(json_extract(data, '$.network.tx_bytes_per_sec')) as net_tx_min,
            MAX(json_extract(data, '$.network.tx_bytes_per_sec')) as net_tx_max
        FROM metrics
        WHERE device_id = :device_id
          AND timestamp >= datetime(:start_ts, 'unixepoch')
          AND timestamp < datetime(:end_ts, 'unixepoch')
        GROUP BY bucket_ts
        ORDER BY bucket_ts
    """
    )

    result = session.execute(
        query,
        {
            "bucket": bucket_seconds,
            "device_id": device_id,
            "start_ts": start_ts,
            "end_ts": end_ts,
        },
    )
    rows = result.fetchall()

    # Build response - now just iterating ~100-300 rows
    return [
        {
            "timestamp": datetime.fromtimestamp(row.bucket_ts, tz=timezone.utc)
            .isoformat()
            .replace("+00:00", "Z"),
            "data": {
                "cpu": {"percent": _agg(row.cpu_avg, row.cpu_min, row.cpu_max)},
                "ram": {"percent": _agg(row.ram_avg, row.ram_min, row.ram_max)},
                "network": {
                    "rx_sec": _agg(row.net_rx_avg, row.net_rx_min, row.net_rx_max),
                    "tx_sec": _agg(row.net_tx_avg, row.net_tx_min, row.net_tx_max),
                },
            },
        }
        for row in rows
    ]


def _agg(avg: float | None, min_: float | None, max_: float | None) -> dict | None:
    if avg is None:
        return None
    return {
        "avg": round(avg, 2),
        "min": round(min_, 2),
        "max": round(max_, 2),
    }


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

    # Store metrics
    metric = Metric(
        device_id=device_id,
        timestamp=payload.timestamp or utc_now(),
        data=payload.metrics,
    )
    session.add(metric)
    session.commit()

    return MetricsResponse(status="ok")


@router.get("/{device_id}/metrics")
def get_metrics(
    device_id: str,
    start: datetime = Query(..., description="Start time (ISO format)"),
    end: datetime = Query(..., description="End time (ISO format)"),
    resolution: str = Query(
        ..., description="Aggregation bucket: 1m, 5m, 15m, 1h, 6h, 1d"
    ),
    session: Session = Depends(get_session),
):
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

    # Validate time range
    if end <= start:
        raise HTTPException(
            status_code=400,
            detail="End time must be after start time",
        )

    return get_aggregated_metrics(session, device_id, start, end, resolution)
