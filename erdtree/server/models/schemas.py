from datetime import datetime, timezone
from typing import Optional
from pydantic import BaseModel, field_serializer


def _serialize_utc_datetime(dt: datetime) -> str:
    """Serialize datetime to ISO 8601 with Z suffix (assumes naive is UTC)."""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.isoformat().replace("+00:00", "Z")


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
class MetricsPushPayload(BaseModel):
    """Payload for pushing metrics - device_id comes from URL path."""

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
    has_sensors: bool = False

    @field_serializer("registered_at", "last_seen_at")
    def serialize_datetime(self, dt: datetime) -> str:
        return _serialize_utc_datetime(dt)


class LatestMetric(BaseModel):
    """Latest metric snapshot with timestamp."""

    timestamp: datetime
    data: dict

    @field_serializer("timestamp")
    def serialize_datetime(self, dt: datetime) -> str:
        return _serialize_utc_datetime(dt)


class DeviceSummary(DeviceResponse):
    """Device info with latest metrics included."""

    latest_metrics: Optional[LatestMetric] = None


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


# Aggregated Sensor Metrics
class AggregatedSensorValue(BaseModel):
    avg: float | None
    min: float | None
    max: float | None


class AggregatedSensorDataPoint(BaseModel):
    timestamp: datetime
    value: AggregatedSensorValue

    @field_serializer("timestamp")
    def serialize_datetime(self, dt: datetime) -> str:
        return _serialize_utc_datetime(dt)


class AggregatedSensorData(BaseModel):
    sensor_id: str
    name: str
    sensor_type: str
    unit: str
    data: list[AggregatedSensorDataPoint]


class AggregatedSensorMetricsResponse(BaseModel):
    sensors: list[AggregatedSensorData]
