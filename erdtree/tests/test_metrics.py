from datetime import datetime, timedelta

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
        f"/api/devices/{device_id}/metrics",
        headers=auth_headers,
        json={
            "metrics": {
                "cpu": {"percent": 45.2},
                "ram": {"percent": 78.5, "used_gb": 8, "total_gb": 16},
                "disk": [
                    {"mount": "/", "percent": 60.0, "used_gb": 100, "total_gb": 200}
                ],
                "network": {"rx_bytes_per_sec": 1000, "tx_bytes_per_sec": 500},
            },
        },
    )
    assert response.status_code == 200
    assert response.json()["status"] == "ok"


def test_push_metrics_no_auth(client: TestClient, device_id: str):
    """Test that pushing metrics without API key fails."""
    response = client.post(
        f"/api/devices/{device_id}/metrics",
        json={
            "metrics": {"cpu": {"percent": 45.2}},
        },
    )
    assert response.status_code == 401


def test_push_metrics_device_not_found(client: TestClient, auth_headers: dict):
    response = client.post(
        "/api/devices/nonexistent-id/metrics",
        headers=auth_headers,
        json={
            "metrics": {"cpu": {"percent": 45.2}},
        },
    )
    assert response.status_code == 404
    assert response.json()["detail"] == "Device not found"


def test_get_metrics_requires_start(client: TestClient, device_id: str):
    """Test that start parameter is required."""
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={"end": "2025-01-02T00:00:00", "resolution": "1h"},
    )
    assert response.status_code == 422


def test_get_metrics_requires_end(client: TestClient, device_id: str):
    """Test that end parameter is required."""
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={"start": "2025-01-01T00:00:00", "resolution": "1h"},
    )
    assert response.status_code == 422


def test_get_metrics_requires_resolution(client: TestClient, device_id: str):
    """Test that resolution parameter is required."""
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={"start": "2025-01-01T00:00:00", "end": "2025-01-02T00:00:00"},
    )
    assert response.status_code == 422


def test_get_metrics_invalid_resolution(client: TestClient, device_id: str):
    """Test that invalid resolution returns 400."""
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T00:00:00",
            "end": "2025-01-02T00:00:00",
            "resolution": "2h",
        },
    )
    assert response.status_code == 400
    assert "Invalid resolution" in response.json()["detail"]


def test_get_metrics_end_before_start(client: TestClient, device_id: str):
    """Test that end before start returns 400."""
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-02T00:00:00",
            "end": "2025-01-01T00:00:00",
            "resolution": "1h",
        },
    )
    assert response.status_code == 400
    assert "End time must be after start time" in response.json()["detail"]


def test_get_metrics_device_not_found(client: TestClient):
    """Test that nonexistent device returns 404."""
    response = client.get(
        "/api/devices/nonexistent-id/metrics",
        params={
            "start": "2025-01-01T00:00:00",
            "end": "2025-01-02T00:00:00",
            "resolution": "1h",
        },
    )
    assert response.status_code == 404


def test_get_metrics_empty_range(client: TestClient, device_id: str):
    """Test that empty time range returns buckets with null values."""
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T00:00:00",
            "end": "2025-01-01T01:00:00",
            "resolution": "1h",
        },
    )
    assert response.status_code == 200
    data = response.json()

    # Should return 1 bucket with null values (00:00-01:00 @ 1h = 1 bucket)
    assert len(data) == 1
    bucket = data[0]
    assert bucket["timestamp"] == "2025-01-01T00:00:00Z"

    # All metrics should have structured null values
    assert bucket["data"]["cpu"]["percent"] == {"avg": None, "min": None, "max": None}
    assert bucket["data"]["ram"]["percent"] == {"avg": None, "min": None, "max": None}
    assert bucket["data"]["network"]["rx_sec"] == {
        "avg": None,
        "min": None,
        "max": None,
    }
    assert bucket["data"]["network"]["tx_sec"] == {
        "avg": None,
        "min": None,
        "max": None,
    }


def test_get_metrics_aggregation(
    client: TestClient, device_id: str, auth_headers: dict
):
    """Test that metrics are correctly aggregated."""
    base_time = datetime(2025, 1, 1, 10, 0, 0)

    # Push metrics with known values within same minute bucket
    for i in range(3):
        timestamp = base_time + timedelta(seconds=i * 10)
        client.post(
            f"/api/devices/{device_id}/metrics",
            headers=auth_headers,
            json={
                "timestamp": timestamp.isoformat(),
                "metrics": {
                    "cpu": {"percent": 10.0 + i * 10, "cores": 4},  # 10, 20, 30
                    "ram": {"percent": 50.0, "total_gb": 16, "used_gb": 8},
                    "network": {
                        "rx_bytes_per_sec": 100 * (i + 1),  # 100, 200, 300
                        "tx_bytes_per_sec": 50,
                    },
                },
            },
        )

    # Query with 1 minute resolution (all should be in one bucket)
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T10:00:00",
            "end": "2025-01-01T10:01:00",
            "resolution": "1m",
        },
    )

    assert response.status_code == 200
    data = response.json()
    assert len(data) == 1

    bucket = data[0]
    assert "timestamp" in bucket
    assert "data" in bucket

    # Check CPU aggregation: avg of 10, 20, 30 = 20
    cpu_percent = bucket["data"]["cpu"]["percent"]
    assert cpu_percent["avg"] == 20.0
    assert cpu_percent["min"] == 10.0
    assert cpu_percent["max"] == 30.0

    # Check RAM aggregation: all 50.0
    ram_percent = bucket["data"]["ram"]["percent"]
    assert ram_percent["avg"] == 50.0
    assert ram_percent["min"] == 50.0
    assert ram_percent["max"] == 50.0

    # Check network aggregation: avg of 100, 200, 300 = 200
    net_rx = bucket["data"]["network"]["rx_sec"]
    assert net_rx["avg"] == 200.0
    assert net_rx["min"] == 100.0
    assert net_rx["max"] == 300.0


def test_get_metrics_bucket_alignment(
    client: TestClient, device_id: str, auth_headers: dict
):
    """Test that bucket timestamps are correctly aligned."""
    # Push metrics at specific times
    times = [
        datetime(2025, 1, 1, 10, 0, 30),  # Should be in 10:00 bucket
        datetime(2025, 1, 1, 10, 5, 15),  # Should be in 10:05 bucket
        datetime(2025, 1, 1, 10, 5, 45),  # Should also be in 10:05 bucket
    ]

    for t in times:
        client.post(
            f"/api/devices/{device_id}/metrics",
            headers=auth_headers,
            json={
                "timestamp": t.isoformat(),
                "metrics": {"cpu": {"percent": 50.0, "cores": 4}},
            },
        )

    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T10:00:00",
            "end": "2025-01-01T10:10:00",
            "resolution": "5m",
        },
    )

    assert response.status_code == 200
    data = response.json()
    assert len(data) == 2  # Two 5-minute buckets

    # Verify bucket timestamps are aligned to 5-minute boundaries
    timestamps = [b["timestamp"] for b in data]
    assert "2025-01-01T10:00:00Z" in timestamps
    assert "2025-01-01T10:05:00Z" in timestamps


def test_timezone_aware_timestamp_normalized(
    client: TestClient, device_id: str, auth_headers: dict
):
    """Ensure timezone-aware timestamps are stored correctly."""
    # Send with explicit timezone
    client.post(
        f"/api/devices/{device_id}/metrics",
        headers=auth_headers,
        json={
            "timestamp": "2025-01-15T10:00:00-05:00",  # EST = 15:00 UTC
            "metrics": {"cpu": {"percent": 50.0}},
        },
    )

    # Query in UTC - should find the metric
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-15T14:00:00Z",  # UTC
            "end": "2025-01-15T16:00:00Z",
            "resolution": "1h",
        },
    )

    assert response.status_code == 200
    data = response.json()

    # Gap-filling returns 2 buckets: 14:00 (null) and 15:00 (with data)
    assert len(data) == 2
    assert data[0]["timestamp"] == "2025-01-15T14:00:00Z"
    assert data[0]["data"]["cpu"]["percent"]["avg"] is None  # No data in 14:00 bucket

    assert data[1]["timestamp"] == "2025-01-15T15:00:00Z"
    assert data[1]["data"]["cpu"]["percent"]["avg"] == 50.0  # Data in 15:00 bucket


def test_gap_filling_sparse_data(
    client: TestClient, device_id: str, auth_headers: dict
):
    """Test that gaps in data are filled with null buckets."""
    # Push metrics at 10:00 and 10:10 (skipping 10:05)
    client.post(
        f"/api/devices/{device_id}/metrics",
        headers=auth_headers,
        json={
            "timestamp": "2025-01-01T10:00:30",
            "metrics": {"cpu": {"percent": 50.0}},
        },
    )
    client.post(
        f"/api/devices/{device_id}/metrics",
        headers=auth_headers,
        json={
            "timestamp": "2025-01-01T10:10:30",
            "metrics": {"cpu": {"percent": 75.0}},
        },
    )

    # Query 10:00-10:15 @ 5m resolution (should return 3 buckets)
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T10:00:00",
            "end": "2025-01-01T10:15:00",
            "resolution": "5m",
        },
    )

    assert response.status_code == 200
    data = response.json()
    assert len(data) == 3  # 10:00, 10:05, 10:10

    # Verify timestamps
    assert data[0]["timestamp"] == "2025-01-01T10:00:00Z"
    assert data[1]["timestamp"] == "2025-01-01T10:05:00Z"
    assert data[2]["timestamp"] == "2025-01-01T10:10:00Z"

    # First bucket has data
    assert data[0]["data"]["cpu"]["percent"]["avg"] == 50.0

    # Second bucket is a gap (null values)
    null_agg = {"avg": None, "min": None, "max": None}
    assert data[1]["data"]["cpu"]["percent"] == null_agg
    assert data[1]["data"]["ram"]["percent"] == null_agg
    assert data[1]["data"]["network"]["rx_sec"] == null_agg
    assert data[1]["data"]["network"]["tx_sec"] == null_agg

    # Third bucket has data
    assert data[2]["data"]["cpu"]["percent"]["avg"] == 75.0


def test_gap_filling_bucket_count(client: TestClient, device_id: str):
    """Test that correct number of buckets are returned for various resolutions."""
    # Query 1 hour @ 5m resolution = 12 buckets
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T10:00:00",
            "end": "2025-01-01T11:00:00",
            "resolution": "5m",
        },
    )
    assert response.status_code == 200
    assert len(response.json()) == 12

    # Query 1 hour @ 15m resolution = 4 buckets
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T10:00:00",
            "end": "2025-01-01T11:00:00",
            "resolution": "15m",
        },
    )
    assert response.status_code == 200
    assert len(response.json()) == 4

    # Query 24 hours @ 1h resolution = 24 buckets
    response = client.get(
        f"/api/devices/{device_id}/metrics",
        params={
            "start": "2025-01-01T00:00:00",
            "end": "2025-01-02T00:00:00",
            "resolution": "1h",
        },
    )
    assert response.status_code == 200
    assert len(response.json()) == 24
