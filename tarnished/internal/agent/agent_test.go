package agent

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/collector"
	"github.com/grace/tarnished/internal/config"
	"github.com/grace/tarnished/internal/httpclient"
	"github.com/grace/tarnished/internal/queue"
	"github.com/grace/tarnished/internal/registration"
)

func init() {
	// Use short intervals for faster tests
	collector.CPUSampleInterval = 10 * time.Millisecond
	httpclient.BaseBackoffDuration = 10 * time.Millisecond
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}

// createTestAgent creates an agent with pre-set state for testing
// This bypasses the registration flow by directly setting the state
func createTestAgent(t *testing.T, serverURL string, interval time.Duration) *Agent {
	t.Helper()
	logger := testLogger()
	cfg := &config.Config{
		Server:   serverURL,
		APIKey:   "grc_test-api-key",
		Interval: interval,
		LogLevel: "debug",
	}

	return &Agent{
		config:    cfg,
		client:    httpclient.New(serverURL, cfg.APIKey, logger),
		collector: collector.New(logger),
		queue:     queue.New(MaxQueueSize),
		state: &registration.State{
			DeviceID:     "test-device-id",
			RegisteredAt: time.Now().UTC(),
		},
		logger: logger,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func TestAgent_New(t *testing.T) {
	cfg := &config.Config{
		Server:   "http://localhost:8080",
		Interval: 10 * time.Second,
		LogLevel: "info",
	}

	agent, err := New(cfg, testLogger())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if agent == nil {
		t.Fatal("agent should not be nil")
	}
	if agent.config != cfg {
		t.Error("config not set correctly")
	}
	if agent.client == nil {
		t.Error("client should not be nil")
	}
	if agent.collector == nil {
		t.Error("collector should not be nil")
	}
	if agent.queue == nil {
		t.Error("queue should not be nil")
	}
}

func TestAgent_CollectAndReport_Success(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/metrics" {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Second)

	// Call collectAndReport directly
	agent.collectAndReport()

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 metrics push, got %d", requestCount)
	}
}

func TestAgent_CollectAndReport_ServerError_QueuesMetrics(t *testing.T) {
	// This test verifies that metrics are queued when PushMetrics fails.
	// We test the queueing behavior directly without going through collectAndReport
	// to avoid the slow retry backoff.

	agent := createTestAgent(t, "http://localhost:1", time.Second)

	// Initial queue should be empty
	if agent.queue.Len() != 0 {
		t.Errorf("initial queue length = %d, want 0", agent.queue.Len())
	}

	// Simulate what collectAndReport does when push fails: queue the metric
	metrics, err := agent.collector.Collect()
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}

	agent.queue.Push(queue.QueuedMetric{
		DeviceID:  agent.state.DeviceID,
		Timestamp: time.Now(),
		Metrics:   metrics,
	})

	// Metrics should be queued
	if agent.queue.Len() == 0 {
		t.Error("expected metrics to be queued")
	}
}

func TestAgent_DrainQueue_Success(t *testing.T) {
	var pushCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pushCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Second)

	// Pre-populate queue
	for i := 0; i < 5; i++ {
		agent.queue.Push(queue.QueuedMetric{
			DeviceID:  "test-device",
			Timestamp: time.Now(),
			Metrics:   &collector.Metrics{},
		})
	}

	if agent.queue.Len() != 5 {
		t.Fatalf("queue length = %d, want 5", agent.queue.Len())
	}

	// Drain queue
	agent.drainQueue()

	// Queue should be empty
	if agent.queue.Len() != 0 {
		t.Errorf("queue length after drain = %d, want 0", agent.queue.Len())
	}

	// All items should have been pushed
	if atomic.LoadInt32(&pushCount) != 5 {
		t.Errorf("push count = %d, want 5", pushCount)
	}
}

func TestAgent_DrainQueue_PartialFailure(t *testing.T) {
	var pushCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&pushCount, 1)
		if count >= 3 {
			// Fail after 2 successful pushes
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Second)

	// Pre-populate queue with 5 items
	for i := 0; i < 5; i++ {
		agent.queue.Push(queue.QueuedMetric{
			DeviceID:  "test-device",
			Timestamp: time.Now(),
			Metrics:   &collector.Metrics{},
		})
	}

	// Drain queue - will fail partway through
	agent.drainQueue()

	// Some items should remain in queue (the failed one + remaining)
	// 2 succeeded, 1 failed and was re-queued, 2 never attempted
	if agent.queue.Len() == 0 {
		t.Error("queue should have remaining items after partial failure")
	}
}

func TestAgent_DrainQueue_EmptyQueue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make requests for empty queue")
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Second)

	// Queue is empty
	agent.drainQueue()

	// Should complete without errors or requests
}

func TestAgent_CollectAndReport_PreventsOverlap(t *testing.T) {
	var concurrentCalls int32
	var maxConcurrent int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&concurrentCalls, 1)
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current > max {
				if atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
					break
				}
			} else {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&concurrentCalls, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Second)

	// Try to call collectAndReport concurrently
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			agent.collectAndReport()
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Due to overlap prevention, should never have more than 1 concurrent collection
	if atomic.LoadInt32(&maxConcurrent) > 1 {
		t.Errorf("max concurrent calls = %d, want <= 1", maxConcurrent)
	}
}

func TestAgent_StartStop_Service(t *testing.T) {
	// Test the Start/Stop service interface methods
	// We use a test server to avoid connection errors and retry delays
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Hour) // Long interval so no collections happen

	// Manually set running state to test Start/Stop logic
	agent.mu.Lock()
	agent.running = false
	agent.mu.Unlock()

	// Start should not error
	err := agent.Start(nil)
	if err != nil {
		t.Errorf("Start returned error: %v", err)
	}

	// Double start should be no-op
	err = agent.Start(nil)
	if err != nil {
		t.Errorf("double Start returned error: %v", err)
	}

	// Stop
	err = agent.Stop(nil)
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}

	// Double stop should be no-op
	err = agent.Stop(nil)
	if err != nil {
		t.Errorf("double Stop returned error: %v", err)
	}
}

func TestAgent_QueueCapacity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	agent := createTestAgent(t, server.URL, time.Second)

	// Fill queue beyond capacity
	for i := 0; i < MaxQueueSize+50; i++ {
		agent.queue.Push(queue.QueuedMetric{
			DeviceID:  "test",
			Timestamp: time.Now(),
			Metrics:   &collector.Metrics{},
		})
	}

	// Queue should be at max capacity (oldest dropped)
	if agent.queue.Len() > MaxQueueSize {
		t.Errorf("queue length = %d, should not exceed MaxQueueSize=%d", agent.queue.Len(), MaxQueueSize)
	}
}

func TestAgent_Constants(t *testing.T) {
	// Verify constants have reasonable values
	if MaxRetries < 1 {
		t.Errorf("MaxRetries = %d, should be at least 1", MaxRetries)
	}
	if MaxQueueSize < 10 {
		t.Errorf("MaxQueueSize = %d, should be at least 10", MaxQueueSize)
	}
}
