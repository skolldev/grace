"""
Performance tests for metrics aggregation endpoint.

These tests validate response times under realistic data volumes.
Run with: pytest tests/performance/ -v -m performance

Performance Targets (from PERF-metrics-aggregation.md):
- 1h @ 1m:   target 50ms,  max 100ms
- 24h @ 5m:  target 100ms, max 200ms
- 7d @ 1h:   target 150ms, max 300ms
- 30d @ 6h:  target 250ms, max 500ms
- 90d @ 1d:  target 400ms, max 800ms
"""

from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import timedelta

import pytest

from .conftest import measure_query

# Mark all tests in this module as performance tests
pytestmark = pytest.mark.performance


# Test scenarios from PERF doc
SCENARIOS = [
    {
        "name": "1h_1m",
        "hours": 1,
        "resolution": "1m",
        "expected_points": 60,
        "target_ms": 50,
        "max_ms": 100,
    },
    {
        "name": "24h_5m",
        "hours": 24,
        "resolution": "5m",
        "expected_points": 288,
        "target_ms": 100,
        "max_ms": 200,
    },
    {
        "name": "7d_1h",
        "hours": 168,
        "resolution": "1h",
        "expected_points": 168,
        "target_ms": 150,
        "max_ms": 300,
    },
    {
        "name": "30d_6h",
        "hours": 720,
        "resolution": "6h",
        "expected_points": 120,
        "target_ms": 250,
        "max_ms": 500,
    },
    {
        "name": "90d_1d",
        "hours": 2160,
        "resolution": "1d",
        "expected_points": 90,
        "target_ms": 400,
        "max_ms": 800,
    },
]


class TestSingleRequestLatency:
    """Test single request latency for various time ranges."""

    @pytest.mark.parametrize(
        "scenario",
        SCENARIOS,
        ids=[s["name"] for s in SCENARIOS],
    )
    def test_latency(self, perf_client, populated_device, scenario):
        """Test that query latency meets performance targets."""
        device_id = populated_device["device_id"]
        end = populated_device["data_end"]
        start = end - timedelta(hours=scenario["hours"])

        # Warmup query to prime SQLite cache
        perf_client.get(
            f"/api/devices/{device_id}/metrics",
            params={
                "start": start.isoformat(),
                "end": end.isoformat(),
                "resolution": scenario["resolution"],
            },
        )

        # Measure performance
        results = measure_query(
            perf_client,
            device_id,
            start,
            end,
            scenario["resolution"],
            iterations=10,
        )

        # Report results
        print(f"\n{scenario['name']} results:")
        print(
            f"  Median: {results['median']:.1f}ms (target: {scenario['target_ms']}ms)"
        )
        print(f"  P95:    {results['p95']:.1f}ms (max: {scenario['max_ms']}ms)")
        print(f"  Range:  {results['min']:.1f}ms - {results['max']:.1f}ms")

        # Assert within acceptable limits
        assert (
            results["p95"] <= scenario["max_ms"]
        ), f"P95 latency {results['p95']:.1f}ms exceeds max {scenario['max_ms']}ms"


class TestColdVsWarm:
    """Test cold vs warm query performance."""

    def test_cold_vs_warm(self, perf_client, populated_device):
        """
        Compare first query (cold cache) vs subsequent queries (warm cache).

        Cold queries may be ~2x slower due to SQLite page cache being empty.
        """
        device_id = populated_device["device_id"]
        end = populated_device["data_end"]
        start = end - timedelta(days=7)

        # Cold query (first query)
        cold_results = measure_query(
            perf_client,
            device_id,
            start,
            end,
            "1h",
            iterations=1,
        )

        # Warm queries (subsequent)
        warm_results = measure_query(
            perf_client,
            device_id,
            start,
            end,
            "1h",
            iterations=10,
        )

        print("\nCold vs Warm (7d @ 1h):")
        print(f"  Cold: {cold_results['median']:.1f}ms")
        print(f"  Warm: {warm_results['median']:.1f}ms (median)")
        print(f"  Ratio: {cold_results['median'] / warm_results['median']:.1f}x")

        # Both should be within acceptable limits
        assert warm_results["p95"] <= 300, "Warm query too slow"


class TestConcurrentLoad:
    """Test concurrent request handling."""

    def test_concurrent_5(self, perf_client, populated_device):
        """Test 5 concurrent requests (typical peak load)."""
        self._run_concurrent_test(
            perf_client,
            populated_device,
            concurrent=5,
            target_p95_ms=300,
            max_p95_ms=500,
        )

    def test_concurrent_10(self, perf_client, populated_device):
        """Test 10 concurrent requests (stress test)."""
        self._run_concurrent_test(
            perf_client,
            populated_device,
            concurrent=10,
            target_p95_ms=500,
            max_p95_ms=1000,
        )

    def _run_concurrent_test(
        self,
        perf_client,
        populated_device,
        concurrent: int,
        target_p95_ms: int,
        max_p95_ms: int,
    ):
        """Run concurrent load test."""
        device_id = populated_device["device_id"]
        end = populated_device["data_end"]
        start = end - timedelta(days=7)

        def make_request():
            """Make a single request and return response time in ms."""
            import time

            t0 = time.perf_counter()
            response = perf_client.get(
                f"/api/devices/{device_id}/metrics",
                params={
                    "start": start.isoformat(),
                    "end": end.isoformat(),
                    "resolution": "1h",
                },
            )
            t1 = time.perf_counter()
            return (t1 - t0) * 1000, response.status_code

        # Run concurrent requests
        times = []
        with ThreadPoolExecutor(max_workers=concurrent) as executor:
            futures = [executor.submit(make_request) for _ in range(concurrent)]
            for future in as_completed(futures):
                time_ms, status = future.result()
                assert status == 200
                times.append(time_ms)

        # Calculate p95
        sorted_times = sorted(times)
        p95_index = int(len(sorted_times) * 0.95)
        p95 = sorted_times[p95_index]

        print(f"\nConcurrent {concurrent} requests (7d @ 1h):")
        print(f"  P95: {p95:.1f}ms (target: {target_p95_ms}ms, max: {max_p95_ms}ms)")
        print(f"  All times: {[f'{t:.0f}' for t in sorted_times]}")

        assert p95 <= max_p95_ms, f"P95 latency {p95:.1f}ms exceeds max {max_p95_ms}ms"


class TestDataPoints:
    """Test that correct number of data points are returned."""

    @pytest.mark.parametrize(
        "scenario",
        SCENARIOS,
        ids=[s["name"] for s in SCENARIOS],
    )
    def test_data_point_count(self, perf_client, populated_device, scenario):
        """Verify aggregation returns expected number of data points."""
        device_id = populated_device["device_id"]
        end = populated_device["data_end"]
        start = end - timedelta(hours=scenario["hours"])

        response = perf_client.get(
            f"/api/devices/{device_id}/metrics",
            params={
                "start": start.isoformat(),
                "end": end.isoformat(),
                "resolution": scenario["resolution"],
            },
        )

        assert response.status_code == 200
        data = response.json()

        # Allow some tolerance for boundary conditions
        expected = scenario["expected_points"]
        actual = len(data)
        tolerance = max(2, int(expected * 0.1))  # 10% or at least 2

        print(f"\n{scenario['name']}: {actual} points (expected ~{expected})")

        assert (
            abs(actual - expected) <= tolerance
        ), f"Expected ~{expected} data points, got {actual}"
