from fastapi.testclient import TestClient


def test_register_device(client: TestClient, auth_headers: dict):
    response = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={
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


def test_register_device_no_auth(client: TestClient):
    """Test that registration without API key fails."""
    response = client.post(
        "/api/devices/register",
        json={
            "hostname": "test-host",
            "os": "linux",
            "arch": "amd64",
        },
    )
    assert response.status_code == 401


def test_register_device_invalid_api_key(client: TestClient):
    """Test that registration with invalid API key fails."""
    response = client.post(
        "/api/devices/register",
        headers={"Authorization": "Bearer invalid-key"},
        json={
            "hostname": "test-host",
            "os": "linux",
            "arch": "amd64",
        },
    )
    assert response.status_code == 401
    assert response.json()["detail"] == "Invalid API key"


def test_register_multiple_devices(client: TestClient, auth_headers: dict):
    """Test that multiple devices can register with the same API key."""
    response1 = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "host1", "os": "linux", "arch": "amd64"},
    )
    response2 = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "host2", "os": "windows", "arch": "amd64"},
    )
    assert response1.status_code == 200
    assert response2.status_code == 200
    assert response1.json()["device_id"] != response2.json()["device_id"]


def test_list_devices_empty(client: TestClient):
    response = client.get("/api/devices")
    assert response.status_code == 200
    assert response.json() == []


def test_list_devices(client: TestClient, auth_headers: dict):
    client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "test-host", "os": "linux", "arch": "amd64"},
    )
    response = client.get("/api/devices")
    assert response.status_code == 200
    devices = response.json()
    assert len(devices) == 1
    assert devices[0]["hostname"] == "test-host"


def test_get_device(client: TestClient, auth_headers: dict):
    reg_response = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "test-host", "os": "linux", "arch": "amd64"},
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


def test_delete_device(client: TestClient, auth_headers: dict):
    reg_response = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "test-host", "os": "linux", "arch": "amd64"},
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
