from datetime import datetime
from typing import Optional
import uuid
from sqlmodel import SQLModel, Field, JSON, Column


class Device(SQLModel, table=True):
    __tablename__ = "devices"

    id: str = Field(default_factory=lambda: str(uuid.uuid4()), primary_key=True)
    hostname: str
    os: str
    arch: str
    ip_address: Optional[str] = None
    registered_at: datetime = Field(default_factory=datetime.utcnow)
    last_seen_at: datetime = Field(default_factory=datetime.utcnow)


class RegistrationToken(SQLModel, table=True):
    __tablename__ = "registration_tokens"

    token: str = Field(default_factory=lambda: str(uuid.uuid4()), primary_key=True)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    expires_at: datetime
    used: bool = Field(default=False)


class Metric(SQLModel, table=True):
    __tablename__ = "metrics"

    id: int = Field(default=None, primary_key=True)
    device_id: str = Field(foreign_key="devices.id", index=True)
    timestamp: datetime = Field(default_factory=datetime.utcnow, index=True)
    data: dict = Field(sa_column=Column(JSON))


class Log(SQLModel, table=True):
    __tablename__ = "logs"

    id: int = Field(default=None, primary_key=True)
    content: str
    type: str = Field(default="info", index=True)  # info, warning, error
    timestamp: datetime = Field(default_factory=datetime.utcnow, index=True)
    source: str = Field(index=True)  # e.g., "devices", "metrics", "admin"
