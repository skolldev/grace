"""
Performance test fixtures for metrics aggregation.

These tests generate large amounts of data to validate query performance
under realistic conditions (up to 778k rows per device over 90 days).
"""

import random
import statistics
import tempfile
import time
from datetime import UTC, datetime, timedelta
from pathlib import Path

import pytest
from fastapi.testclient import TestClient
from sqlmodel import Session, SQLModel, create_engine

from server.core import auth
from server.core.database import get_session, set_engine_override
from server.main import app
from server.models.models import (
    Device,
    DeviceSensor,
    Log,
    Metric,
    Setting,
)  # noqa: F401


def generate_test_metrics(
    session: Session,
    device_id: str,
    days: int,
    end_time: datetime,
    interval_seconds: int = 10,
) -> int:
    """
    Generate realistic metric data for testing.

    Args:
        session: Database session
        device_id: Device to generate metrics for
        days: Number of days of data to generate
        interval_seconds: Interval between data points (default 10s)

    Returns:
        Number of rows inserted
    """
    # Set seed for reproducible test data
    random.seed(42)

    end = end_time
    start = end - timedelta(days=days)
    current = start

    batch = []
    row_count = 0

    while current < end:
        metric = Metric(
            device_id=device_id,
            timestamp=current,
            cpu_percent=random.uniform(5, 95),
            ram_percent=random.uniform(30, 80),
            ram_used_gb=random.uniform(5, 13),
            ram_total_gb=16,
            disk=[
                {
                    "mount": "/",
                    "percent": random.uniform(10, 90),
                    "used_gb": random.uniform(50, 450),
                    "total_gb": 500,
                }
            ],
            net_rx_bytes_sec=random.randint(100, 10000),
            net_tx_bytes_sec=random.randint(100, 5000),
        )
        batch.append(metric)
        row_count += 1

        # Batch insert every 10,000 rows for efficiency
        if len(batch) >= 10000:
            session.bulk_save_objects(batch)
            session.commit()
            batch = []

        current += timedelta(seconds=interval_seconds)

    # Insert remaining rows
    if batch:
        session.bulk_save_objects(batch)
        session.commit()

    return row_count


def measure_query(
    client: TestClient,
    device_id: str,
    start: datetime,
    end: datetime,
    resolution: str,
    iterations: int = 10,
) -> dict:
    """
    Measure query performance over multiple iterations.

    Args:
        client: Test client
        device_id: Device to query
        start: Start time
        end: End time
        resolution: Aggregation resolution
        iterations: Number of iterations to run

    Returns:
        Dict with min, max, mean, median, p95 in milliseconds
    """
    times = []

    for _ in range(iterations):
        t0 = time.perf_counter()
        response = client.get(
            f"/api/devices/{device_id}/metrics",
            params={
                "start": start.isoformat(),
                "end": end.isoformat(),
                "resolution": resolution,
            },
        )
        t1 = time.perf_counter()

        assert response.status_code == 200
        times.append((t1 - t0) * 1000)  # Convert to ms

    sorted_times = sorted(times)

    # Calculate P95 correctly (handles small sample sizes)
    if len(sorted_times) == 1:
        p95 = sorted_times[0]
    else:
        # P95 is at position 0.95 * (N-1) in sorted list
        p95_index = int(0.95 * (len(sorted_times) - 1))
        p95 = sorted_times[p95_index]

    return {
        "min": round(min(times), 2),
        "max": round(max(times), 2),
        "mean": round(statistics.mean(times), 2),
        "median": round(statistics.median(times), 2),
        "p95": round(p95, 2),
        "times": times,
    }


@pytest.fixture(scope="module")
def perf_db_path():
    """Create a temporary file for the performance test database."""
    with tempfile.TemporaryDirectory() as tmpdir:
        yield Path(tmpdir) / "perf_test.db"


@pytest.fixture(scope="module")
def perf_engine(perf_db_path):
    """Create a file-based SQLite engine for realistic I/O performance."""
    engine = create_engine(
        f"sqlite:///{perf_db_path}",
        connect_args={"check_same_thread": False},
    )
    SQLModel.metadata.create_all(engine)
    yield engine
    engine.dispose()


@pytest.fixture(scope="module")
def perf_session(perf_engine):
    """Session scoped to module for data reuse across tests."""
    set_engine_override(perf_engine)
    with Session(perf_engine) as session:
        yield session
    set_engine_override(None)


@pytest.fixture(scope="module")
def perf_client(perf_engine):
    """Test client using the performance database."""
    auth.reset_api_key_cache()
    set_engine_override(perf_engine)

    def get_session_override():
        # Create NEW session for each request (thread-safe)
        with Session(perf_engine) as session:
            yield session

    app.dependency_overrides[get_session] = get_session_override
    client = TestClient(app)
    yield client
    app.dependency_overrides.clear()
    auth.reset_api_key_cache()
    set_engine_override(None)


@pytest.fixture(scope="module")
def perf_api_key(perf_client):
    """Get API key for authenticated requests."""
    response = perf_client.get("/api/admin/key")
    return response.json()["api_key"]


@pytest.fixture(scope="module")
def perf_auth_headers(perf_api_key):
    """Authorization headers for authenticated requests."""
    return {"Authorization": f"Bearer {perf_api_key}"}


@pytest.fixture(scope="module")
def populated_device(perf_client, perf_session, perf_auth_headers):
    """
    Device with 90 days of metrics data (~778k rows).

    This fixture generates a substantial amount of data on first use.
    The data is reused across all tests in the module.
    """
    # Register a test device
    response = perf_client.post(
        "/api/devices/register",
        headers=perf_auth_headers,
        json={"hostname": "perf-test-device", "os": "linux", "arch": "amd64"},
    )
    device_id = response.json()["device_id"]

    end_time = datetime.now(UTC).replace(tzinfo=None)
    # Generate 90 days of metrics data
    print(f"\nGenerating 90 days of metrics data for device {device_id}...")
    row_count = generate_test_metrics(
        perf_session, device_id, days=90, end_time=end_time
    )
    print(f"Generated {row_count:,} rows")

    return {
        "device_id": device_id,
        "row_count": row_count,
        "data_start": end_time - timedelta(days=90),
        "data_end": end_time,
    }
