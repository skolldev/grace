import pytest
from fastapi.testclient import TestClient
from sqlmodel import SQLModel, Session, create_engine
from sqlmodel.pool import StaticPool

from server.main import app
from server.core.database import get_session, set_engine_override
from server.core import auth

# Import all models to ensure they're registered before create_all()
from server.models.models import (
    Device,
    Metric,
    Log,
    Setting,
    DeviceSensor,
    SensorMetric,
)  # noqa: F401


@pytest.fixture(name="session")
def session_fixture():
    engine = create_engine(
        "sqlite://",
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
    SQLModel.metadata.create_all(engine)
    # Override the engine so logger and other components use test database
    set_engine_override(engine)
    with Session(engine) as session:
        yield session
    set_engine_override(None)


@pytest.fixture(name="client")
def client_fixture(session: Session):
    # Reset API key cache for test isolation
    auth.reset_api_key_cache()

    def get_session_override():
        return session

    app.dependency_overrides[get_session] = get_session_override
    client = TestClient(app)
    yield client
    app.dependency_overrides.clear()
    auth.reset_api_key_cache()


@pytest.fixture(name="api_key")
def api_key_fixture(client: TestClient) -> str:
    """Get API key from admin endpoint."""
    response = client.get("/api/admin/key")
    return response.json()["api_key"]


@pytest.fixture(name="auth_headers")
def auth_headers_fixture(api_key: str) -> dict:
    """Return Authorization headers for authenticated requests."""
    return {"Authorization": f"Bearer {api_key}"}
