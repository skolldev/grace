package queue

import (
	"sync"
	"time"
)

const DefaultMaxSize = 100

type QueuedMetric struct {
	DeviceID  string
	Timestamp time.Time
	Metrics   any
}

type MetricQueue struct {
	mu      sync.Mutex
	items   []QueuedMetric
	maxSize int
}

func New(maxSize int) *MetricQueue {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	return &MetricQueue{
		items:   make([]QueuedMetric, 0, maxSize),
		maxSize: maxSize,
	}
}

func (q *MetricQueue) Push(m QueuedMetric) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) >= q.maxSize {
		// Drop oldest
		q.items = q.items[1:]
	}
	q.items = append(q.items, m)
}

func (q *MetricQueue) Pop() (QueuedMetric, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return QueuedMetric{}, false
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *MetricQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *MetricQueue) Drain() []QueuedMetric {
	q.mu.Lock()
	defer q.mu.Unlock()

	items := q.items
	q.items = make([]QueuedMetric, 0, q.maxSize)
	return items
}
