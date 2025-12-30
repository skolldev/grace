package sensors

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConfigRefreshInterval is the duration between sensor config refreshes
const ConfigRefreshInterval = 5 * time.Minute

// Manager handles sensor discovery and data collection
type Manager struct {
	reader      *Reader
	enabledSet  map[string]bool
	lastRefresh time.Time
	mu          sync.RWMutex
	logger      *zap.Logger
	available   bool
}

// NewManager creates a new sensor manager
func NewManager(logger *zap.Logger) *Manager {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     logger,
	}

	reader, err := NewReader()
	if err != nil {
		logger.Warn("HWiNFO not available, sensors disabled", zap.Error(err))
		m.available = false
	} else {
		m.reader = reader
		m.available = true
	}

	return m
}

// Available returns true if HWiNFO is accessible
func (m *Manager) Available() bool {
	return m.available
}

// Close releases resources
func (m *Manager) Close() error {
	if m.reader != nil {
		return m.reader.Close()
	}
	return nil
}

// DiscoverSensors reads all available sensors
func (m *Manager) DiscoverSensors() ([]SensorEntry, error) {
	if !m.available {
		return nil, nil
	}
	return m.reader.Read()
}

// UpdateConfig updates which sensors are enabled
func (m *Manager) UpdateConfig(enabled []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabledSet = make(map[string]bool, len(enabled))
	for _, id := range enabled {
		m.enabledSet[id] = true
	}
	m.lastRefresh = time.Now()
	m.logger.Debug("sensor config updated", zap.Int("enabled_count", len(enabled)))
}

// ShouldRefreshConfig returns true if config should be refreshed
func (m *Manager) ShouldRefreshConfig() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.lastRefresh) >= ConfigRefreshInterval
}

// CollectEnabled reads only the enabled sensors
func (m *Manager) CollectEnabled() (map[string]float64, error) {
	if !m.available {
		return nil, nil
	}

	entries, err := m.reader.Read()
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]float64)
	for _, entry := range entries {
		if m.enabledSet[entry.ID] {
			result[entry.ID] = entry.Value
		}
	}

	return result, nil
}
