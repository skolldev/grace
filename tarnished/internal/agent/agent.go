package agent

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/collector"
	"github.com/grace/tarnished/internal/config"
	"github.com/grace/tarnished/internal/httpclient"
	"github.com/grace/tarnished/internal/queue"
	"github.com/grace/tarnished/internal/registration"
)

const maxRetries = 5

type Agent struct {
	config    *config.Config
	client    *httpclient.Client
	collector *collector.Collector
	queue     *queue.MetricQueue
	state     *registration.State
	logger    *zap.Logger
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func New(cfg *config.Config, logger *zap.Logger) (*Agent, error) {
	client := httpclient.New(cfg.Server, logger)
	coll := collector.New(logger)
	q := queue.New(100)

	return &Agent{
		config:    cfg,
		client:    client,
		collector: coll,
		queue:     q,
		logger:    logger,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}, nil
}

// Start implements service.Interface
func (a *Agent) Start(s service.Service) error {
	a.logger.Info("starting agent")
	go a.run()
	return nil
}

// Stop implements service.Interface
func (a *Agent) Stop(s service.Service) error {
	a.logger.Info("stopping agent")
	close(a.stopCh)
	<-a.doneCh
	return nil
}

// RunForeground runs the agent in foreground mode (blocking)
func (a *Agent) RunForeground() error {
	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go a.run()

	<-sigCh
	a.logger.Info("received shutdown signal")
	close(a.stopCh)
	<-a.doneCh

	return nil
}

func (a *Agent) run() {
	defer close(a.doneCh)

	// Check registration
	state, err := registration.LoadState()
	if err != nil {
		a.logger.Error("failed to load state", zap.Error(err))
		return
	}

	if state == nil {
		// First run - need to register
		if a.config.Token == "" {
			a.logger.Error("no device_id found and no token provided - cannot register")
			return
		}

		a.logger.Info("registering device")
		state, err = registration.Register(a.client, a.config.Token)
		if err != nil {
			a.logger.Error("registration failed", zap.Error(err))
			return
		}
		a.logger.Info("device registered successfully", zap.String("device_id", state.DeviceID))
	}

	a.state = state
	a.logger.Info("agent running",
		zap.String("device_id", state.DeviceID),
		zap.Duration("interval", a.config.Interval))

	ticker := time.NewTicker(a.config.Interval)
	defer ticker.Stop()

	// Initial collection
	a.collectAndReport()

	for {
		select {
		case <-ticker.C:
			a.collectAndReport()
		case <-a.stopCh:
			a.logger.Info("agent stopped")
			return
		}
	}
}

func (a *Agent) collectAndReport() {
	// First, try to drain any queued metrics
	a.drainQueue()

	// Collect new metrics
	metrics, err := a.collector.Collect()
	if err != nil {
		a.logger.Error("failed to collect metrics", zap.Error(err))
		return
	}

	timestamp := time.Now().UTC()

	// Try to push with retry
	err = a.client.PushMetricsWithRetry(a.state.DeviceID, timestamp, metrics, maxRetries)
	if err != nil {
		a.logger.Warn("failed to push metrics, queueing", zap.Error(err))
		a.queue.Push(queue.QueuedMetric{
			DeviceID:  a.state.DeviceID,
			Timestamp: timestamp,
			Metrics:   metrics,
		})
		a.logger.Info("queued metric", zap.Int("queue_size", a.queue.Len()))
	} else {
		a.logger.Debug("metrics pushed successfully")
	}
}

func (a *Agent) drainQueue() {
	queueLen := a.queue.Len()
	if queueLen == 0 {
		return
	}

	a.logger.Info("draining queued metrics", zap.Int("count", queueLen))

	for {
		item, ok := a.queue.Pop()
		if !ok {
			break
		}

		metrics, ok := item.Metrics.(*collector.Metrics)
		if !ok {
			a.logger.Warn("invalid metric type in queue")
			continue
		}

		err := a.client.PushMetrics(item.DeviceID, item.Timestamp, metrics)
		if err != nil {
			// Re-queue on failure and stop draining
			a.logger.Warn("failed to push queued metric, re-queueing", zap.Error(err))
			a.queue.Push(item)
			return
		}
	}

	a.logger.Info("successfully drained all queued metrics")
}
