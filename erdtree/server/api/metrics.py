from datetime import datetime
from typing import Optional
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlmodel import Session, select

from server.core.auth import verify_api_key
from server.core.database import get_session
from server.models.models import Device, Metric, utc_now
from server.models.schemas import MetricsPayload, MetricsResponse

router = APIRouter()


@router.post("", response_model=MetricsResponse)
def push_metrics(
    payload: MetricsPayload,
    session: Session = Depends(get_session),
    _: None = Depends(verify_api_key),
):
    # Verify device exists
    device = session.get(Device, payload.device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Update last seen
    device.last_seen_at = utc_now()
    session.add(device)

    # Store metrics
    metric = Metric(
        device_id=payload.device_id,
        timestamp=payload.timestamp or utc_now(),
        data=payload.metrics,
    )
    session.add(metric)
    session.commit()

    return MetricsResponse(status="ok")


@router.get("/{device_id}")
def get_metrics(
    device_id: str,
    start: Optional[datetime] = Query(None, description="Start time filter"),
    end: Optional[datetime] = Query(None, description="End time filter"),
    limit: int = Query(100, ge=1, le=1000),
    session: Session = Depends(get_session),
):
    # Verify device exists
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    query = select(Metric).where(Metric.device_id == device_id)

    if start:
        query = query.where(Metric.timestamp >= start)
    if end:
        query = query.where(Metric.timestamp <= end)

    query = query.order_by(Metric.timestamp.desc()).limit(limit)

    metrics = session.exec(query).all()

    return [
        {
            "timestamp": m.timestamp.isoformat(),
            "data": m.data,
        }
        for m in metrics
    ]
