from fastapi import APIRouter, Depends, HTTPException
from sqlmodel import Session, select

from server.core.auth import verify_api_key
from server.core.database import get_session
from server.core.logger import log_info
from server.models.models import Device, DeviceSensor
from server.models.schemas import (
    ReportSensorsRequest,
    ReportSensorsResponse,
    DeviceSensorResponse,
    UpdateSensorConfigRequest,
    SensorConfigResponse,
)

router = APIRouter()


@router.post("/{device_id}/sensors", response_model=ReportSensorsResponse)
def report_sensors(
    device_id: str,
    request: ReportSensorsRequest,
    session: Session = Depends(get_session),
    _: None = Depends(verify_api_key),
):
    """Agent reports available sensors for discovery."""
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Upsert sensors - update existing, add new
    for sensor_info in request.sensors:
        existing = session.get(DeviceSensor, (device_id, sensor_info.sensor_id))
        if existing:
            # Update metadata but preserve enabled status and display_name
            existing.name = sensor_info.name
            existing.sensor_type = sensor_info.sensor_type
            existing.unit = sensor_info.unit
            existing.source = sensor_info.source
            session.add(existing)
        else:
            # New sensor - insert with enabled=False
            new_sensor = DeviceSensor(
                device_id=device_id,
                sensor_id=sensor_info.sensor_id,
                name=sensor_info.name,
                sensor_type=sensor_info.sensor_type,
                unit=sensor_info.unit,
                source=sensor_info.source,
                enabled=False,
            )
            session.add(new_sensor)

    # Set has_sensors flag on device if not already set
    if not device.has_sensors:
        device.has_sensors = True
        session.add(device)

    session.commit()
    log_info(f"Device {device_id} reported {len(request.sensors)} sensors", "sensors")

    return ReportSensorsResponse(status="ok", count=len(request.sensors))


@router.get("/{device_id}/sensors", response_model=list[DeviceSensorResponse])
def get_sensors(device_id: str, session: Session = Depends(get_session)):
    """UI fetches available sensors for a device."""
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    sensors = session.exec(
        select(DeviceSensor).where(DeviceSensor.device_id == device_id)
    ).all()

    return sensors


@router.put("/{device_id}/sensors/config", response_model=SensorConfigResponse)
def update_sensor_config(
    device_id: str,
    request: UpdateSensorConfigRequest,
    session: Session = Depends(get_session),
):
    """UI updates which sensors are enabled."""
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    # Get all sensors for device
    sensors = session.exec(
        select(DeviceSensor).where(DeviceSensor.device_id == device_id)
    ).all()

    enabled_set = set(request.enabled)

    for sensor in sensors:
        sensor.enabled = sensor.sensor_id in enabled_set
        session.add(sensor)

    session.commit()
    log_info(
        f"Device {device_id} sensor config updated: {len(enabled_set)} enabled",
        "sensors",
    )

    return SensorConfigResponse(enabled=request.enabled)


@router.get("/{device_id}/sensors/config", response_model=SensorConfigResponse)
def get_sensor_config(
    device_id: str,
    session: Session = Depends(get_session),
    _: None = Depends(verify_api_key),
):
    """Agent fetches its sensor config (which sensors to track)."""
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    sensors = session.exec(
        select(DeviceSensor)
        .where(DeviceSensor.device_id == device_id)
        .where(DeviceSensor.enabled)
    ).all()

    return SensorConfigResponse(enabled=[s.sensor_id for s in sensors])


@router.delete("/{device_id}/sensors")
def delete_sensors(device_id: str, session: Session = Depends(get_session)):
    """Delete all sensors for a device."""
    device = session.get(Device, device_id)
    if not device:
        raise HTTPException(status_code=404, detail="Device not found")

    sensors = session.exec(
        select(DeviceSensor).where(DeviceSensor.device_id == device_id)
    ).all()

    count = len(sensors)
    for s in sensors:
        session.delete(s)

    session.commit()
    log_info(f"Deleted {count} sensors for device {device_id}", "sensors")

    return {"status": "deleted", "count": count}
