from datetime import datetime
from typing import Optional
from pydantic import BaseModel


# Registration
class RegisterRequest(BaseModel):
    token: str
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
class TokenResponse(BaseModel):
    token: str
    expires_at: datetime