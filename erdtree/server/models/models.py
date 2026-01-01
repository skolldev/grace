from datetime import UTC, datetime
from typing import Optional
import uuid
from sqlmodel import SQLModel, Field, JSON, Column, Index


def utc_now() -> datetime:
    """Return current UTC time as a naive datetime (for SQLite compatibility)."""
    return datetime.now(UTC).replace(tzinfo=None)


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


class Log(SQLModel, table=True):
    __tablename__ = "logs"

    id: int = Field(default=None, primary_key=True)
    content: str
    type: str = Field(default="info", index=True)  # info, warning, error
    timestamp: datetime = Field(default_factory=utc_now, index=True)
    source: str = Field(index=True)  # e.g., "devices", "metrics", "admin"


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
