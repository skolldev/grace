import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def device_id(client: TestClient, auth_headers: dict) -> str:
    reg_response = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "test-host", "os": "linux", "arch": "amd64"},
    )
    return reg_response.json()["device_id"]


def test_push_metrics(client: TestClient, device_id: str, auth_headers: dict):
    response = client.post(
        "/api/metrics",
        headers=auth_headers,
        json={
            "device_id": device_id,
            "metrics": {"cpu": 45.2, "memory": 78.5, "disk": 60.0},
        },
    )
    assert response.status_code == 200
    assert response.json()["status"] == "ok"


def test_push_metrics_no_auth(client: TestClient, device_id: str):
    """Test that pushing metrics without API key fails."""
    response = client.post(
        "/api/metrics",
        json={
            "device_id": device_id,
            "metrics": {"cpu": 45.2},
        },
    )
    assert response.status_code == 401


def test_push_metrics_device_not_found(client: TestClient, auth_headers: dict):
    response = client.post(
        "/api/metrics",
        headers=auth_headers,
        json={
            "device_id": "nonexistent-id",
            "metrics": {"cpu": 45.2},
        },
    )
    assert response.status_code == 404
    assert response.json()["detail"] == "Device not found"


def test_get_metrics(client: TestClient, device_id: str, auth_headers: dict):
    # Push some metrics
    client.post(
        "/api/metrics",
        headers=auth_headers,
        json={"device_id": device_id, "metrics": {"cpu": 10.0}},
    )
    client.post(
        "/api/metrics",
        headers=auth_headers,
        json={"device_id": device_id, "metrics": {"cpu": 20.0}},
    )

    response = client.get(f"/api/metrics/{device_id}")
    assert response.status_code == 200
    metrics = response.json()
    assert len(metrics) == 2


def test_get_metrics_empty(client: TestClient, device_id: str):
    response = client.get(f"/api/metrics/{device_id}")
    assert response.status_code == 200
    assert response.json() == []


def test_get_metrics_device_not_found(client: TestClient):
    response = client.get("/api/metrics/nonexistent-id")
    assert response.status_code == 404


def test_get_metrics_with_limit(client: TestClient, device_id: str, auth_headers: dict):
    for i in range(5):
        client.post(
            "/api/metrics",
            headers=auth_headers,
            json={"device_id": device_id, "metrics": {"cpu": i}},
        )

    response = client.get(f"/api/metrics/{device_id}?limit=3")
    assert response.status_code == 200
    metrics = response.json()
    assert len(metrics) == 3
