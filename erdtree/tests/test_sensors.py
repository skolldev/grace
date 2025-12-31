import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def registered_device(client: TestClient, auth_headers: dict) -> str:
    """Create a registered device and return its ID."""
    reg_response = client.post(
        "/api/devices/register",
        headers=auth_headers,
        json={"hostname": "test-host", "os": "windows", "arch": "amd64"},
    )
    return reg_response.json()["device_id"]


def test_report_sensors(client: TestClient, registered_device: str, auth_headers: dict):
    response = client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu_package",
                    "name": "CPU Package",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
                {
                    "sensor_id": "hwinfo:fan:cpu_fan",
                    "name": "CPU Fan",
                    "sensor_type": "fan",
                    "unit": "RPM",
                    "source": "hwinfo",
                },
            ]
        },
    )
    assert response.status_code == 200
    assert response.json()["status"] == "ok"
    assert response.json()["count"] == 2


def test_report_sensors_no_auth(client: TestClient, registered_device: str):
    """Test that reporting sensors without API key fails."""
    response = client.post(
        f"/api/devices/{registered_device}/sensors",
        json={"sensors": []},
    )
    assert response.status_code == 401


def test_report_sensors_device_not_found(client: TestClient, auth_headers: dict):
    response = client.post(
        "/api/devices/nonexistent/sensors",
        headers=auth_headers,
        json={"sensors": []},
    )
    assert response.status_code == 404


def test_get_sensors(client: TestClient, registered_device: str, auth_headers: dict):
    # First report sensors
    client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu_package",
                    "name": "CPU Package",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                }
            ]
        },
    )

    # Then get sensors (UI endpoint - no auth needed)
    response = client.get(f"/api/devices/{registered_device}/sensors")
    assert response.status_code == 200
    sensors = response.json()
    assert len(sensors) == 1
    assert sensors[0]["sensor_id"] == "hwinfo:temp:cpu_package"
    assert sensors[0]["enabled"] is False  # Default


def test_get_sensors_device_not_found(client: TestClient):
    response = client.get("/api/devices/nonexistent/sensors")
    assert response.status_code == 404


def test_update_sensor_config(
    client: TestClient, registered_device: str, auth_headers: dict
):
    # Report sensors first
    client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu",
                    "name": "CPU",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
                {
                    "sensor_id": "hwinfo:temp:gpu",
                    "name": "GPU",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
            ]
        },
    )

    # Enable one sensor (UI endpoint - no auth needed)
    response = client.put(
        f"/api/devices/{registered_device}/sensors/config",
        json={"enabled": ["hwinfo:temp:cpu"]},
    )
    assert response.status_code == 200
    assert response.json()["enabled"] == ["hwinfo:temp:cpu"]

    # Verify state
    sensors = client.get(f"/api/devices/{registered_device}/sensors").json()
    cpu_sensor = next(s for s in sensors if s["sensor_id"] == "hwinfo:temp:cpu")
    gpu_sensor = next(s for s in sensors if s["sensor_id"] == "hwinfo:temp:gpu")
    assert cpu_sensor["enabled"] is True
    assert gpu_sensor["enabled"] is False


def test_update_sensor_config_device_not_found(client: TestClient):
    response = client.put(
        "/api/devices/nonexistent/sensors/config",
        json={"enabled": []},
    )
    assert response.status_code == 404


def test_get_sensor_config(
    client: TestClient, registered_device: str, auth_headers: dict
):
    # Report and enable sensors
    client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu",
                    "name": "CPU",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
            ]
        },
    )
    client.put(
        f"/api/devices/{registered_device}/sensors/config",
        json={"enabled": ["hwinfo:temp:cpu"]},
    )

    # Get config (agent endpoint - requires auth)
    response = client.get(
        f"/api/devices/{registered_device}/sensors/config",
        headers=auth_headers,
    )
    assert response.status_code == 200
    assert "hwinfo:temp:cpu" in response.json()["enabled"]


def test_get_sensor_config_no_auth(client: TestClient, registered_device: str):
    """Test that getting sensor config without API key fails."""
    response = client.get(f"/api/devices/{registered_device}/sensors/config")
    assert response.status_code == 401


def test_get_sensor_config_device_not_found(client: TestClient, auth_headers: dict):
    response = client.get(
        "/api/devices/nonexistent/sensors/config",
        headers=auth_headers,
    )
    assert response.status_code == 404


def test_delete_device_cascades_sensors(
    client: TestClient, registered_device: str, auth_headers: dict
):
    # Report sensors
    client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu",
                    "name": "CPU",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
            ]
        },
    )

    # Delete device
    response = client.delete(f"/api/devices/{registered_device}")
    assert response.status_code == 200

    # Device and sensors should be gone
    response = client.get(f"/api/devices/{registered_device}/sensors")
    assert response.status_code == 404


def test_report_sensors_upsert(
    client: TestClient, registered_device: str, auth_headers: dict
):
    """Re-reporting sensors should update metadata but preserve enabled status."""
    # Initial report
    client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu",
                    "name": "CPU Original",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
            ]
        },
    )

    # Enable the sensor
    client.put(
        f"/api/devices/{registered_device}/sensors/config",
        json={"enabled": ["hwinfo:temp:cpu"]},
    )

    # Re-report with updated name
    client.post(
        f"/api/devices/{registered_device}/sensors",
        headers=auth_headers,
        json={
            "sensors": [
                {
                    "sensor_id": "hwinfo:temp:cpu",
                    "name": "CPU Updated",
                    "sensor_type": "temperature",
                    "unit": "C",
                    "source": "hwinfo",
                },
            ]
        },
    )

    # Verify: name updated, enabled preserved
    sensors = client.get(f"/api/devices/{registered_device}/sensors").json()
    assert sensors[0]["name"] == "CPU Updated"
    assert sensors[0]["enabled"] is True
