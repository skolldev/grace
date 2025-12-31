package agent

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/collector"
	"github.com/grace/tarnished/internal/config"
	"github.com/grace/tarnished/internal/httpclient"
	"github.com/grace/tarnished/internal/queue"
	"github.com/grace/tarnished/internal/registration"
	"github.com/grace/tarnished/internal/sensors"
)

const (
	MaxRetries   = 5
	MaxQueueSize = 100
)

type Agent struct {
	config    *config.Config
	client    *httpclient.Client
	collector *collector.Collector
	queue     *queue.MetricQueue
	state     *registration.State
	logger    *zap.Logger
	stopCh    chan struct{}
	doneCh    chan struct{}
	sensors   *sensors.Manager

	mu         sync.Mutex
	running    bool
	collecting bool
}

func New(cfg *config.Config, logger *zap.Logger) (*Agent, error) {
	client := httpclient.New(cfg.Server, cfg.APIKey, logger)
	coll := collector.New(logger)
	q := queue.New(MaxQueueSize)

	var sensorMgr *sensors.Manager
	if cfg.SensorsEnabled {
		sensorMgr = sensors.NewManager(logger)
	}

	return &Agent{
		config:    cfg,
		client:    client,
		collector: coll,
		queue:     q,
		sensors:   sensorMgr,
		logger:    logger,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}, nil
}

// Start implements service.Interface
func (a *Agent) Start(s service.Service) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = true
	a.stopCh = make(chan struct{})
	a.doneCh = make(chan struct{})
	a.mu.Unlock()

	a.logger.Info("starting agent")
	go a.run()
	return nil
}

// Stop implements service.Interface
func (a *Agent) Stop(s service.Service) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	a.mu.Unlock()

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
		a.logger.Info("registering device")
		state, err = registration.Register(a.client)
		if err != nil {
			a.logger.Error("registration failed", zap.Error(err))
			return
		}
		a.logger.Info("device registered successfully", zap.String("device_id", state.DeviceID))
	}

	a.state = state

	// Initialize sensors if enabled
	if a.sensors != nil && a.sensors.Available() {
		a.initSensors()
	}

	a.logger.Info("agent running",
		zap.String("device_id", state.DeviceID),
		zap.Duration("interval", a.config.Interval),
		zap.Bool("sensors_enabled", a.sensors != nil && a.sensors.Available()))

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
			if a.sensors != nil {
				a.sensors.Close()
			}
			return
		}
	}
}

// initSensors discovers sensors and reports them to the server
func (a *Agent) initSensors() {
	entries, err := a.sensors.DiscoverSensors()
	if err != nil {
		a.logger.Warn("failed to discover sensors", zap.Error(err))
		return
	}

	if len(entries) == 0 {
		a.logger.Info("no sensors discovered")
		return
	}

	// Log discovered sensors
	a.logger.Info("discovered sensors", zap.Int("count", len(entries)))
	for _, e := range entries {
		a.logger.Info("sensor",
			zap.String("id", e.ID),
			zap.String("name", e.Name),
			zap.String("type", e.SensorType),
			zap.String("unit", e.Unit),
			zap.Float64("value", e.Value))
	}

	// Convert to API format and report
	sensorInfos := make([]httpclient.SensorInfo, len(entries))
	for i, e := range entries {
		sensorInfos[i] = httpclient.SensorInfo{
			SensorID:   e.ID,
			Name:       e.Name,
			SensorType: e.SensorType,
			Unit:       e.Unit,
			Source:     e.Source,
		}
	}

	if err := a.client.ReportSensors(a.state.DeviceID, sensorInfos); err != nil {
		a.logger.Warn("failed to report sensors to server", zap.Error(err))
	} else {
		a.logger.Info("reported sensors to server", zap.Int("count", len(entries)))
	}

	// Fetch initial config
	a.refreshSensorConfig()
}

// refreshSensorConfig fetches the sensor config from the server
func (a *Agent) refreshSensorConfig() {
	enabled, err := a.client.GetSensorConfig(a.state.DeviceID)
	if err != nil {
		a.logger.Warn("failed to fetch sensor config", zap.Error(err))
		return
	}
	a.sensors.UpdateConfig(enabled)
}

func (a *Agent) collectAndReport() {
	// Prevent overlapping collections
	a.mu.Lock()
	if a.collecting {
		a.mu.Unlock()
		a.logger.Debug("skipping collection, previous collection still running")
		return
	}
	a.collecting = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.collecting = false
		a.mu.Unlock()
	}()

	// First, try to drain any queued metrics
	a.drainQueue()

	// Check if sensor config needs refresh
	if a.sensors != nil && a.sensors.Available() && a.sensors.ShouldRefreshConfig() {
		a.refreshSensorConfig()
	}

	// Collect new metrics
	metrics, err := a.collector.Collect()
	if err != nil {
		a.logger.Error("failed to collect metrics", zap.Error(err))
		return
	}

	// Collect sensor data if enabled
	if a.sensors != nil && a.sensors.Available() {
		sensorData, err := a.sensors.CollectEnabled()
		if err != nil {
			a.logger.Warn("failed to collect sensor data", zap.Error(err))
		} else if len(sensorData) > 0 {
			metrics.Sensors = sensorData
		}
	}

	timestamp := time.Now().UTC()

	// Try to push with retry
	err = a.client.PushMetricsWithRetry(a.state.DeviceID, timestamp, metrics, MaxRetries)
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

		err := a.client.PushMetrics(item.DeviceID, item.Timestamp, item.Metrics)
		if err != nil {
			// Re-queue on failure and stop draining
			a.logger.Warn("failed to push queued metric, re-queueing", zap.Error(err))
			a.queue.Push(item)
			return
		}
	}

	a.logger.Info("successfully drained all queued metrics")
}
