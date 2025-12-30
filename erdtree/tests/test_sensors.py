import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def registered_device(client: TestClient) -> str:
    """Create a registered device and return its ID."""
    token_response = client.post("/api/admin/tokens")
    token = token_response.json()["token"]

    reg_response = client.post(
        "/api/devices/register",
        json={"token": token, "hostname": "test-host", "os": "windows", "arch": "amd64"},
    )
    return reg_response.json()["device_id"]


def test_report_sensors(client: TestClient, registered_device: str):
    response = client.post(
        f"/api/devices/{registered_device}/sensors",
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


def test_report_sensors_device_not_found(client: TestClient):
    response = client.post(
        "/api/devices/nonexistent/sensors",
        json={"sensors": []},
    )
    assert response.status_code == 404


def test_get_sensors(client: TestClient, registered_device: str):
    # First report sensors
    client.post(
        f"/api/devices/{registered_device}/sensors",
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

    # Then get sensors
    response = client.get(f"/api/devices/{registered_device}/sensors")
    assert response.status_code == 200
    sensors = response.json()
    assert len(sensors) == 1
    assert sensors[0]["sensor_id"] == "hwinfo:temp:cpu_package"
    assert sensors[0]["enabled"] is False  # Default


def test_get_sensors_device_not_found(client: TestClient):
    response = client.get("/api/devices/nonexistent/sensors")
    assert response.status_code == 404


def test_update_sensor_config(client: TestClient, registered_device: str):
    # Report sensors first
    client.post(
        f"/api/devices/{registered_device}/sensors",
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

    # Enable one sensor
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


def test_get_sensor_config(client: TestClient, registered_device: str):
    # Report and enable sensors
    client.post(
        f"/api/devices/{registered_device}/sensors",
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

    # Get config
    response = client.get(f"/api/devices/{registered_device}/sensors/config")
    assert response.status_code == 200
    assert "hwinfo:temp:cpu" in response.json()["enabled"]


def test_get_sensor_config_device_not_found(client: TestClient):
    response = client.get("/api/devices/nonexistent/sensors/config")
    assert response.status_code == 404


def test_delete_device_cascades_sensors(client: TestClient, registered_device: str):
    # Report sensors
    client.post(
        f"/api/devices/{registered_device}/sensors",
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


def test_report_sensors_upsert(client: TestClient, registered_device: str):
    """Re-reporting sensors should update metadata but preserve enabled status."""
    # Initial report
    client.post(
        f"/api/devices/{registered_device}/sensors",
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
