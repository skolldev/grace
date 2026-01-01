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

    # SQLite: unixepoch() for timestamp conversion, integer division for bucketing
    # Use unixepoch() for all comparisons to handle datetime format differences
    query = text("""
        SELECT
            (unixepoch(timestamp) / :bucket) * :bucket as bucket_ts,
            json_extract(data, '$.cpu.percent') as cpu_percent,
            json_extract(data, '$.ram.percent') as ram_percent,
            json_extract(data, '$.network.rx_bytes_per_sec') as net_rx,
            json_extract(data, '$.network.tx_bytes_per_sec') as net_tx
        FROM metrics
        WHERE device_id = :device_id
          AND unixepoch(timestamp) >= unixepoch(:start)
          AND unixepoch(timestamp) < unixepoch(:end)
        ORDER BY timestamp
    """)

    result = session.execute(
        query,
        {
            "bucket": bucket_seconds,
            "device_id": device_id,
            "start": start.isoformat(),
            "end": end.isoformat(),
        },
    )
    rows = result.fetchall()

    # Group by bucket and compute aggregates in Python
    buckets: dict[int, dict[str, list]] = {}
    for row in rows:
        bucket_ts = row.bucket_ts
        if bucket_ts not in buckets:
            buckets[bucket_ts] = {
                "cpu_percent": [],
                "ram_percent": [],
                "net_rx": [],
                "net_tx": [],
            }
        if row.cpu_percent is not None:
            buckets[bucket_ts]["cpu_percent"].append(row.cpu_percent)
        if row.ram_percent is not None:
            buckets[bucket_ts]["ram_percent"].append(row.ram_percent)
        if row.net_rx is not None:
            buckets[bucket_ts]["net_rx"].append(row.net_rx)
        if row.net_tx is not None:
            buckets[bucket_ts]["net_tx"].append(row.net_tx)

    # Build response
    aggregated = []
    for bucket_ts in sorted(buckets.keys()):
        b = buckets[bucket_ts]
        aggregated.append(
            {
                "timestamp": datetime.fromtimestamp(bucket_ts, tz=timezone.utc)
                .isoformat()
                .replace("+00:00", "Z"),
                "data": {
                    "cpu": {"percent": aggregate(b["cpu_percent"])},
                    "ram": {"percent": aggregate(b["ram_percent"])},
                    "network": {
                        "rx_sec": aggregate(b["net_rx"]),
                        "tx_sec": aggregate(b["net_tx"]),
                    },
                },
            }
        )

    return aggregated


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
