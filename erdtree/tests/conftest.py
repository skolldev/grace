import pytest
from fastapi.testclient import TestClient
from sqlmodel import SQLModel, Session, create_engine
from sqlmodel.pool import StaticPool

from server.main import app
from server.core.database import get_session
from server.core import auth


@pytest.fixture(name="session")
def session_fixture():
    engine = create_engine(
        "sqlite://",
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
    SQLModel.metadata.create_all(engine)
    with Session(engine) as session:
        yield session


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
