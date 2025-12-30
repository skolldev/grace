from server.core.logger import log_info
from fastapi import APIRouter, Depends, HTTPException
from sqlmodel import Session, select

from server.core.database import get_session
from server.models.models import Device, Metric, RegistrationToken, utc_now
from server.models.schemas import (
    RegisterRequest,
    RegisterResponse,
    DeviceResponse,
    DeviceWithMetrics,
)

router = APIRouter()


@router.post("/register", response_model=RegisterResponse)
def register_device(request: RegisterRequest, session: Session = Depends(get_session)):
    # Validate token
    token = session.get(RegistrationToken, request.token)
    if not token:
        raise HTTPException(status_code=400, detail="Invalid token")
    if token.used:
        raise HTTPException(status_code=400, detail="Token already used")
    if token.expires_at < utc_now():
        raise HTTPException(status_code=400, detail="Token expired")

    # Mark token as used
    token.used = True
    session.add(token)

    # Create device
    device = Device(
        hostname=request.hostname,
        os=request.os,
        arch=request.arch,
        ip_address=request.ip_address,
    )
    session.add(device)
    session.commit()
    session.refresh(device)
    log_info(f"Device registered successfully: {device.id}", "system")
    return RegisterResponse(
        device_id=device.id, message="Device registered successfully"
    )


@router.get("", response_model=list[DeviceResponse])
def list_devices(session: Session = Depends(get_session)):
    devices = session.exec(select(Device)).all()
    return devices


@router.get("/{device_id}", response_model=DeviceWithMetrics)
def get_device(device_id: str, session: Session = Depends(get_session)):
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Get latest metric
    latest = session.exec(
        select(Metric)
        .where(Metric.device_id == device_id)
        .order_by(Metric.timestamp.desc())
        .limit(1)
    ).first()

    return DeviceWithMetrics(
        **device.model_dump(),
        latest_metrics=latest.data if latest else None,
    )


@router.delete("/{device_id}")
def delete_device(device_id: str, session: Session = Depends(get_session)):
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Delete associated metrics
    metrics = session.exec(select(Metric).where(Metric.device_id == device_id)).all()
    for m in metrics:
        session.delete(m)

    session.delete(device)
    session.commit()

    return {"status": "deleted"}
