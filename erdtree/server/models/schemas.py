from datetime import datetime
from typing import Optional
from pydantic import BaseModel


# Registration
class RegisterRequest(BaseModel):
    hostname: str
    os: str
    arch: str
    ip_address: Optional[str] = None


class RegisterResponse(BaseModel):
    device_id: str
    message: str


# Metrics
class MetricsPayload(BaseModel):
    device_id: str
    timestamp: Optional[datetime] = None
    metrics: dict


class MetricsResponse(BaseModel):
    status: str


# Devices
class DeviceResponse(BaseModel):
    id: str
    hostname: str
    os: str
    arch: str
    ip_address: Optional[str]
    registered_at: datetime
    last_seen_at: datetime


class DeviceWithMetrics(DeviceResponse):
    latest_metrics: Optional[dict] = None


# Admin
class ApiKeyResponse(BaseModel):
    api_key: str


# Sensors
class SensorInfo(BaseModel):
    sensor_id: str  # hwinfo:temp:cpu_package
    name: str  # Original HWiNFO name
    sensor_type: str  # temperature, voltage, fan, etc.
    unit: str  # C, V, RPM, etc.
    source: str  # "hwinfo"


class ReportSensorsRequest(BaseModel):
    sensors: list[SensorInfo]


class ReportSensorsResponse(BaseModel):
    status: str
    count: int


class DeviceSensorResponse(BaseModel):
    device_id: str
    sensor_id: str
    name: str
    display_name: Optional[str]
    sensor_type: str
    unit: str
    enabled: bool
    source: str


class UpdateSensorConfigRequest(BaseModel):
    enabled: list[str]  # List of sensor_ids to enable


class SensorConfigResponse(BaseModel):
    enabled: list[str]  # List of enabled sensor_ids
