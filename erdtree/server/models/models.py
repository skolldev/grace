from datetime import UTC, datetime, timezone
from typing import Optional
import uuid
from pydantic import field_serializer
from sqlmodel import SQLModel, Field, JSON, Column, Index


def utc_now() -> datetime:
    """Return current UTC time as a naive datetime (for SQLite compatibility)."""
    return datetime.now(UTC).replace(tzinfo=None)


def serialize_utc_datetime(dt: datetime) -> str:
    """Serialize datetime to ISO 8601 with Z suffix (assumes naive is UTC)."""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.isoformat().replace("+00:00", "Z")


def to_naive_utc(dt: datetime | None) -> datetime | None:
    """
    Convert any datetime to naive UTC.

    - If None, returns None
    - If naive, assumes it's already UTC
    - If aware, converts to UTC then strips tzinfo
    """
    if dt is None:
        return None
    if dt.tzinfo is None:
        return dt  # Assume naive means UTC
    return dt.astimezone(timezone.utc).replace(tzinfo=None)


class Setting(SQLModel, table=True):
    __tablename__ = "settings"

    key: str = Field(primary_key=True)
    value: str


class Device(SQLModel, table=True):
    __tablename__ = "devices"

    id: str = Field(default_factory=lambda: str(uuid.uuid4()), primary_key=True)
    hostname: str
    os: str
    arch: str
    ip_address: Optional[str] = None
    registered_at: datetime = Field(default_factory=utc_now)
    last_seen_at: datetime = Field(default_factory=utc_now)
    has_sensors: bool = Field(default=False)

    @field_serializer("registered_at", "last_seen_at")
    def serialize_datetime(self, dt: datetime) -> str:
        return serialize_utc_datetime(dt)


class Metric(SQLModel, table=True):
    __tablename__ = "metrics"

    id: int = Field(default=None, primary_key=True)
    device_id: str = Field(foreign_key="devices.id", index=True)
    timestamp: datetime = Field(default_factory=utc_now, index=True)

    # CPU
    cpu_percent: Optional[float] = None

    # RAM
    ram_percent: Optional[float] = None
    ram_used_gb: Optional[float] = None
    ram_total_gb: Optional[float] = None

    # Disk (keep as JSON - agent sends array of partitions)
    disk: Optional[list] = Field(default=None, sa_column=Column(JSON))

    # Network
    net_rx_bytes_sec: Optional[int] = None
    net_tx_bytes_sec: Optional[int] = None

    __table_args__ = (Index("idx_metric_device_timestamp", "device_id", "timestamp"),)

    @field_serializer("timestamp")
    def serialize_datetime(self, dt: datetime) -> str:
        return serialize_utc_datetime(dt)


class Log(SQLModel, table=True):
    __tablename__ = "logs"

    id: int = Field(default=None, primary_key=True)
    content: str
    type: str = Field(default="info", index=True)  # info, warning, error
    timestamp: datetime = Field(default_factory=utc_now, index=True)
    source: str = Field(index=True)  # e.g., "devices", "metrics", "admin"

    @field_serializer("timestamp")
    def serialize_datetime(self, dt: datetime) -> str:
        return serialize_utc_datetime(dt)


class DeviceSensor(SQLModel, table=True):
    __tablename__ = "device_sensors"

    device_id: str = Field(foreign_key="devices.id", primary_key=True)
    sensor_id: str = Field(primary_key=True)  # e.g., "hwinfo:temp:cpu_package"
    name: str  # Original name from HWiNFO
    display_name: Optional[str] = None  # User-customized name
    sensor_type: str  # temperature, voltage, fan, etc.
    unit: str  # C, V, RPM, etc.
    enabled: bool = Field(default=False)
    source: str  # "hwinfo"


class SensorMetric(SQLModel, table=True):
    __tablename__ = "sensor_metrics"

    id: int = Field(default=None, primary_key=True)
    device_id: str = Field(foreign_key="devices.id", index=True)
    sensor_id: str = Field(index=True)
    timestamp: datetime = Field(default_factory=utc_now, index=True)
    value: float

    __table_args__ = (
        Index("idx_sensor_metric_device_sensor_ts", "device_id", "sensor_id", "timestamp"),
    )

    @field_serializer("timestamp")
    def serialize_datetime(self, dt: datetime) -> str:
        return serialize_utc_datetime(dt)
