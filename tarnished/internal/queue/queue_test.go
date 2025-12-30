package queue

import (
	"sync"
	"testing"
	"time"

	"github.com/grace/tarnished/internal/collector"
)

func TestMetricQueue_New(t *testing.T) {
	tests := []struct {
		name        string
		maxSize     int
		wantMaxSize int
	}{
		{"positive capacity", 10, 10},
		{"zero uses default", 0, DefaultMaxSize},
		{"negative uses default", -5, DefaultMaxSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New(tt.maxSize)
			if q.maxSize != tt.wantMaxSize {
				t.Errorf("maxSize = %d, want %d", q.maxSize, tt.wantMaxSize)
			}
			if q.Len() != 0 {
				t.Errorf("initial length = %d, want 0", q.Len())
			}
		})
	}
}

func TestMetricQueue_PushPop(t *testing.T) {
	q := New(10)

	// Test empty queue
	_, ok := q.Pop()
	if ok {
		t.Error("Pop on empty queue should return false")
	}

	// Push and pop single item
	item := QueuedMetric{
		DeviceID:  "device-1",
		Timestamp: time.Now(),
		Metrics:   &collector.Metrics{},
	}
	q.Push(item)

	if q.Len() != 1 {
		t.Errorf("Len after push = %d, want 1", q.Len())
	}

	popped, ok := q.Pop()
	if !ok {
		t.Error("Pop should return true for non-empty queue")
	}
	if popped.DeviceID != item.DeviceID {
		t.Errorf("DeviceID = %s, want %s", popped.DeviceID, item.DeviceID)
	}
	if q.Len() != 0 {
		t.Errorf("Len after pop = %d, want 0", q.Len())
	}
}

func TestMetricQueue_FIFO(t *testing.T) {
	q := New(10)

	items := []QueuedMetric{
		{DeviceID: "first", Timestamp: time.Now(), Metrics: &collector.Metrics{}},
		{DeviceID: "second", Timestamp: time.Now(), Metrics: &collector.Metrics{}},
		{DeviceID: "third", Timestamp: time.Now(), Metrics: &collector.Metrics{}},
	}

	for _, item := range items {
		q.Push(item)
	}

	// Should pop in FIFO order
	for i, want := range items {
		got, ok := q.Pop()
		if !ok {
			t.Fatalf("Pop %d should return true", i)
		}
		if got.DeviceID != want.DeviceID {
			t.Errorf("Pop %d: DeviceID = %s, want %s", i, got.DeviceID, want.DeviceID)
		}
	}
}

func TestMetricQueue_CapacityDropsOldest(t *testing.T) {
	capacity := 3
	q := New(capacity)

	// Push more items than capacity
	for i := 0; i < 5; i++ {
		q.Push(QueuedMetric{
			DeviceID:  string(rune('A' + i)), // A, B, C, D, E
			Timestamp: time.Now(),
			Metrics:   &collector.Metrics{},
		})
	}

	// Queue should be at capacity
	if q.Len() != capacity {
		t.Errorf("Len = %d, want %d", q.Len(), capacity)
	}

	// Oldest items (A, B) should be dropped, remaining: C, D, E
	expected := []string{"C", "D", "E"}
	for i, want := range expected {
		got, ok := q.Pop()
		if !ok {
			t.Fatalf("Pop %d should return true", i)
		}
		if got.DeviceID != want {
			t.Errorf("Pop %d: DeviceID = %s, want %s", i, got.DeviceID, want)
		}
	}
}

func TestMetricQueue_EmptyAfterDrain(t *testing.T) {
	q := New(5)

	q.Push(QueuedMetric{DeviceID: "test", Timestamp: time.Now(), Metrics: &collector.Metrics{}})
	q.Pop()

	if q.Len() != 0 {
		t.Errorf("Len after drain = %d, want 0", q.Len())
	}

	_, ok := q.Pop()
	if ok {
		t.Error("Pop on drained queue should return false")
	}
}

func TestMetricQueue_ConcurrentAccess(t *testing.T) {
	q := New(1000)
	var wg sync.WaitGroup
	numGoroutines := 10
	itemsPerGoroutine := 100

	// Concurrent pushes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				q.Push(QueuedMetric{
					DeviceID:  "device",
					Timestamp: time.Now(),
					Metrics:   &collector.Metrics{},
				})
			}
		}(i)
	}
	wg.Wait()

	// All items should be present (capacity is large enough)
	expectedLen := numGoroutines * itemsPerGoroutine
	if q.Len() != expectedLen {
		t.Errorf("Len after concurrent pushes = %d, want %d", q.Len(), expectedLen)
	}

	// Concurrent pops
	popCount := 0
	var popMu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, ok := q.Pop()
				if !ok {
					return
				}
				popMu.Lock()
				popCount++
				popMu.Unlock()
			}
		}()
	}
	wg.Wait()

	if popCount != expectedLen {
		t.Errorf("total popped = %d, want %d", popCount, expectedLen)
	}
	if q.Len() != 0 {
		t.Errorf("Len after concurrent pops = %d, want 0", q.Len())
	}
}

func TestMetricQueue_ConcurrentPushPop(t *testing.T) {
	q := New(50)
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Start multiple pushers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					q.Push(QueuedMetric{
						DeviceID:  "device",
						Timestamp: time.Now(),
						Metrics:   &collector.Metrics{},
					})
				}
			}
		}()
	}

	// Start multiple poppers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					q.Pop()
				}
			}
		}()
	}

	// Run for a short time
	time.Sleep(100 * time.Millisecond)
	close(done)
	wg.Wait()

	// No panic means thread-safe operations succeeded
	// Queue should have valid state
	if q.Len() < 0 || q.Len() > 50 {
		t.Errorf("invalid queue length: %d", q.Len())
	}
}
