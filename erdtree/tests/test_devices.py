import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def valid_token(client: TestClient) -> str:
    response = client.post("/api/admin/tokens")
    return response.json()["token"]


def test_register_device(client: TestClient, valid_token: str):
    response = client.post(
        "/api/devices/register",
        json={
            "token": valid_token,
            "hostname": "test-host",
            "os": "linux",
            "arch": "amd64",
            "ip_address": "192.168.1.100",
        },
    )
    assert response.status_code == 200
    data = response.json()
    assert "device_id" in data
    assert data["message"] == "Device registered successfully"


def test_register_device_invalid_token(client: TestClient):
    response = client.post(
        "/api/devices/register",
        json={
            "token": "invalid-token",
            "hostname": "test-host",
            "os": "linux",
            "arch": "amd64",
        },
    )
    assert response.status_code == 400
    assert response.json()["detail"] == "Invalid token"


def test_register_device_token_already_used(client: TestClient, valid_token: str):
    # First registration
    client.post(
        "/api/devices/register",
        json={"token": valid_token, "hostname": "host1", "os": "linux", "arch": "amd64"},
    )
    # Second registration with same token
    response = client.post(
        "/api/devices/register",
        json={"token": valid_token, "hostname": "host2", "os": "linux", "arch": "amd64"},
    )
    assert response.status_code == 400
    assert response.json()["detail"] == "Token already used"


def test_list_devices_empty(client: TestClient):
    response = client.get("/api/devices")
    assert response.status_code == 200
    assert response.json() == []


def test_list_devices(client: TestClient, valid_token: str):
    client.post(
        "/api/devices/register",
        json={"token": valid_token, "hostname": "test-host", "os": "linux", "arch": "amd64"},
    )
    response = client.get("/api/devices")
    assert response.status_code == 200
    devices = response.json()
    assert len(devices) == 1
    assert devices[0]["hostname"] == "test-host"


def test_get_device(client: TestClient, valid_token: str):
    reg_response = client.post(
        "/api/devices/register",
        json={"token": valid_token, "hostname": "test-host", "os": "linux", "arch": "amd64"},
    )
    device_id = reg_response.json()["device_id"]

    response = client.get(f"/api/devices/{device_id}")
    assert response.status_code == 200
    data = response.json()
    assert data["id"] == device_id
    assert data["hostname"] == "test-host"
    assert data["latest_metrics"] is None


def test_get_device_not_found(client: TestClient):
    response = client.get("/api/devices/nonexistent-id")
    assert response.status_code == 404
    assert response.json()["detail"] == "Device not found"


def test_delete_device(client: TestClient, valid_token: str):
    reg_response = client.post(
        "/api/devices/register",
        json={"token": valid_token, "hostname": "test-host", "os": "linux", "arch": "amd64"},
    )
    device_id = reg_response.json()["device_id"]

    response = client.delete(f"/api/devices/{device_id}")
    assert response.status_code == 200
    assert response.json()["status"] == "deleted"

    # Verify device is gone
    response = client.get(f"/api/devices/{device_id}")
    assert response.status_code == 404


def test_delete_device_not_found(client: TestClient):
    response = client.delete("/api/devices/nonexistent-id")
    assert response.status_code == 404
