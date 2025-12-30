from fastapi.testclient import TestClient
from sqlmodel import Session

from server.models.models import Log


def test_get_logs_empty(client: TestClient):
    response = client.get("/api/logs/")
    assert response.status_code == 200
    assert response.json() == []


def test_get_logs(client: TestClient, session: Session):
    # Add logs directly to the session
    log1 = Log(content="Test log 1", type="info", source="test")
    log2 = Log(content="Test log 2", type="error", source="test")
    session.add(log1)
    session.add(log2)
    session.commit()

    response = client.get("/api/logs/")
    assert response.status_code == 200
    logs = response.json()
    assert len(logs) == 2


def test_get_logs_order(client: TestClient, session: Session):
    log1 = Log(content="First", type="info", source="test")
    session.add(log1)
    session.commit()

    log2 = Log(content="Second", type="info", source="test")
    session.add(log2)
    session.commit()

    response = client.get("/api/logs/")
    logs = response.json()
    # Most recent first
    assert logs[0]["content"] == "Second"
    assert logs[1]["content"] == "First"


def test_health_check(client: TestClient):
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}
