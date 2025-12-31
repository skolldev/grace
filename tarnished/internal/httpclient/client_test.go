package httpclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/collector"
)

func init() {
	// Use a short backoff duration for faster tests
	BaseBackoffDuration = 10 * time.Millisecond
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func TestClient_Register_Success(t *testing.T) {
	expectedDeviceID := "device-123"
	testAPIKey := "grc_test-api-key"
	var receivedReq RegisterRequest
	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/devices/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		receivedAuthHeader = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RegisterResponse{
			DeviceID: expectedDeviceID,
			Message:  "registered",
		})
	}))
	defer server.Close()

	client := New(server.URL, testAPIKey, testLogger())
	req := &RegisterRequest{
		Hostname:  "test-host",
		OS:        "linux",
		Arch:      "amd64",
		IPAddress: "192.168.1.100",
	}

	resp, err := client.Register(req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if resp.DeviceID != expectedDeviceID {
		t.Errorf("DeviceID = %s, want %s", resp.DeviceID, expectedDeviceID)
	}

	// Verify Authorization header was sent
	expectedAuth := "Bearer " + testAPIKey
	if receivedAuthHeader != expectedAuth {
		t.Errorf("Authorization = %s, want %s", receivedAuthHeader, expectedAuth)
	}

	// Verify request was sent correctly
	if receivedReq.Hostname != req.Hostname {
		t.Errorf("Hostname = %s, want %s", receivedReq.Hostname, req.Hostname)
	}
}

func TestClient_Register_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	_, err := client.Register(&RegisterRequest{Hostname: "test"})

	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestClient_Register_InvalidAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid API key"))
	}))
	defer server.Close()

	client := New(server.URL, "bad-key", testLogger())
	_, err := client.Register(&RegisterRequest{Hostname: "test"})

	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestClient_Register_ConnectionFailure(t *testing.T) {
	client := New("http://localhost:99999", "grc_test-key", testLogger())
	_, err := client.Register(&RegisterRequest{Hostname: "test"})

	if err == nil {
		t.Error("expected error for connection failure")
	}
}

func TestClient_PushMetrics_Success(t *testing.T) {
	var receivedPayload MetricsPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/metrics" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MetricsResponse{Status: "ok"})
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	deviceID := "device-123"
	timestamp := time.Now().UTC()
	metrics := &collector.Metrics{
		CPU: collector.CPUMetrics{Percent: 45.5, Cores: 8},
		RAM: collector.RAMMetrics{TotalGB: 16.0, UsedGB: 8.5, Percent: 53.1},
	}

	err := client.PushMetrics(deviceID, timestamp, metrics)
	if err != nil {
		t.Fatalf("PushMetrics failed: %v", err)
	}

	if receivedPayload.DeviceID != deviceID {
		t.Errorf("DeviceID = %s, want %s", receivedPayload.DeviceID, deviceID)
	}
	if receivedPayload.Metrics.CPU.Percent != metrics.CPU.Percent {
		t.Errorf("CPU.Percent = %f, want %f", receivedPayload.Metrics.CPU.Percent, metrics.CPU.Percent)
	}
}

func TestClient_PushMetrics_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("server unavailable"))
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	err := client.PushMetrics("device-1", time.Now(), &collector.Metrics{})

	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestClient_PushMetricsWithRetry_SuccessFirstTry(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	err := client.PushMetricsWithRetry("device-1", time.Now(), &collector.Metrics{}, 5)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&attemptCount) != 1 {
		t.Errorf("attempt count = %d, want 1", attemptCount)
	}
}

func TestClient_PushMetricsWithRetry_SuccessAfterRetries(t *testing.T) {
	var attemptCount int32
	failUntil := int32(3) // Fail first 2 attempts

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count < failUntil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	err := client.PushMetricsWithRetry("device-1", time.Now(), &collector.Metrics{}, 5)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&attemptCount) != failUntil {
		t.Errorf("attempt count = %d, want %d", attemptCount, failUntil)
	}
}

func TestClient_PushMetricsWithRetry_FailsAfterMaxRetries(t *testing.T) {
	var attemptCount int32
	maxRetries := 3

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	err := client.PushMetricsWithRetry("device-1", time.Now(), &collector.Metrics{}, maxRetries)

	if err == nil {
		t.Error("expected error after max retries")
	}
	if atomic.LoadInt32(&attemptCount) != int32(maxRetries) {
		t.Errorf("attempt count = %d, want %d", attemptCount, maxRetries)
	}
}

func TestClient_New(t *testing.T) {
	client := New("http://example.com", "grc_test-key", testLogger())

	if client.baseURL != "http://example.com" {
		t.Errorf("baseURL = %s, want http://example.com", client.baseURL)
	}
	if client.apiKey != "grc_test-key" {
		t.Errorf("apiKey = %s, want grc_test-key", client.apiKey)
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", client.httpClient.Timeout, DefaultTimeout)
	}
}

func TestClient_PushMetricsWithRetry_BackoffTiming(t *testing.T) {
	// Save and restore original backoff for this test
	originalBackoff := BaseBackoffDuration
	BaseBackoffDuration = 50 * time.Millisecond
	defer func() { BaseBackoffDuration = originalBackoff }()

	var timestamps []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		timestamps = append(timestamps, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New(server.URL, "grc_test-key", testLogger())
	start := time.Now()
	_ = client.PushMetricsWithRetry("device-1", time.Now(), &collector.Metrics{}, 4)
	totalDuration := time.Since(start)

	// With 4 retries and backoffs of 50ms, 100ms, 200ms (exponential), expect ~350ms minimum
	// Allow some tolerance for timing
	expectedMin := 300 * time.Millisecond
	if totalDuration < expectedMin {
		t.Errorf("total duration %v is less than expected minimum %v (backoff not applied?)", totalDuration, expectedMin)
	}

	// Verify exponential backoff pattern by checking intervals between attempts
	mu.Lock()
	defer mu.Unlock()
	if len(timestamps) < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", len(timestamps))
	}

	// Check that later intervals are longer (exponential)
	interval1 := timestamps[1].Sub(timestamps[0])
	interval2 := timestamps[2].Sub(timestamps[1])

	// Second interval should be roughly 2x the first (with tolerance)
	if interval2 < interval1 {
		t.Errorf("backoff not exponential: interval1=%v, interval2=%v", interval1, interval2)
	}
}
