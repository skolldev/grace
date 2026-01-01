from server.core.logger import log_info
from fastapi import APIRouter, Depends, HTTPException
from sqlmodel import Session, select

from server.core.auth import verify_api_key
from server.core.database import get_session
from server.models.models import Device, DeviceSensor, Metric
from server.models.schemas import (
    RegisterRequest,
    RegisterResponse,
    DeviceSummary,
    LatestMetric,
)

router = APIRouter()


def _metric_to_data(metric: Metric) -> dict:
    """Reconstruct the nested data dict from metric columns."""
    return {
        "cpu": {"percent": metric.cpu_percent},
        "ram": {
            "percent": metric.ram_percent,
            "used_gb": metric.ram_used_gb,
            "total_gb": metric.ram_total_gb,
        },
        "disk": metric.disk,  # Array stored as-is
        "network": {
            "rx_bytes_per_sec": metric.net_rx_bytes_sec,
            "tx_bytes_per_sec": metric.net_tx_bytes_sec,
        },
    }


@router.post("/register", response_model=RegisterResponse)
def register_device(
    request: RegisterRequest,
    session: Session = Depends(get_session),
    _: None = Depends(verify_api_key),
):
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


@router.get("", response_model=list[DeviceSummary])
def list_devices(session: Session = Depends(get_session)):
    devices = session.exec(select(Device)).all()

    result = []
    for device in devices:
        latest = session.exec(
            select(Metric)
            .where(Metric.device_id == device.id)
            .order_by(Metric.timestamp.desc())
            .limit(1)
        ).first()

        result.append(
            DeviceSummary(
                **device.model_dump(),
                latest_metrics=LatestMetric(
                    timestamp=latest.timestamp, data=_metric_to_data(latest)
                )
                if latest
                else None,
            )
        )

    return result


@router.get("/{device_id}", response_model=DeviceSummary)
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

    return DeviceSummary(
        **device.model_dump(),
        latest_metrics=LatestMetric(
            timestamp=latest.timestamp, data=_metric_to_data(latest)
        )
        if latest
        else None,
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

    # Delete associated sensors
    device_sensors = session.exec(
        select(DeviceSensor).where(DeviceSensor.device_id == device_id)
    ).all()
    for s in device_sensors:
        session.delete(s)

    session.delete(device)
    session.commit()

    return {"status": "deleted"}
