package sensors

import (
	"testing"

	"go.uber.org/zap"
)

func TestManager_New(t *testing.T) {
	logger := zap.NewNop()
	m := NewManager(logger)

	if m == nil {
		t.Fatal("manager should not be nil")
	}

	// On non-Windows or without HWiNFO, Available should be false
	// This test just verifies graceful handling
}

func TestManager_ConfigUpdate(t *testing.T) {
	logger := zap.NewNop()
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     logger,
	}

	m.UpdateConfig([]string{"hwinfo:temp:cpu", "hwinfo:fan:cpu"})

	if !m.enabledSet["hwinfo:temp:cpu"] {
		t.Error("expected hwinfo:temp:cpu to be enabled")
	}
	if !m.enabledSet["hwinfo:fan:cpu"] {
		t.Error("expected hwinfo:fan:cpu to be enabled")
	}
	if m.enabledSet["hwinfo:temp:gpu"] {
		t.Error("expected hwinfo:temp:gpu to be disabled")
	}
}

func TestManager_ShouldRefreshConfig(t *testing.T) {
	logger := zap.NewNop()
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     logger,
	}

	// Should refresh immediately since lastRefresh is zero
	if !m.ShouldRefreshConfig() {
		t.Error("expected ShouldRefreshConfig to return true initially")
	}

	// After update, should not need refresh
	m.UpdateConfig([]string{})
	if m.ShouldRefreshConfig() {
		t.Error("expected ShouldRefreshConfig to return false after update")
	}
}

func TestManager_Available_WhenReaderNil(t *testing.T) {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     zap.NewNop(),
		available:  false,
		reader:     nil,
	}

	if m.Available() {
		t.Error("expected Available to return false when reader is nil")
	}
}

func TestManager_Close_WhenReaderNil(t *testing.T) {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     zap.NewNop(),
		reader:     nil,
	}

	// Should not panic or error when reader is nil
	err := m.Close()
	if err != nil {
		t.Errorf("Close with nil reader should not error: %v", err)
	}
}

func TestManager_DiscoverSensors_WhenUnavailable(t *testing.T) {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     zap.NewNop(),
		available:  false,
		reader:     nil,
	}

	sensors, err := m.DiscoverSensors()
	if err != nil {
		t.Errorf("DiscoverSensors when unavailable should not error: %v", err)
	}
	if sensors != nil {
		t.Error("expected nil sensors when unavailable")
	}
}

func TestManager_CollectEnabled_WhenUnavailable(t *testing.T) {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     zap.NewNop(),
		available:  false,
		reader:     nil,
	}

	result, err := m.CollectEnabled()
	if err != nil {
		t.Errorf("CollectEnabled when unavailable should not error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when unavailable")
	}
}

func TestManager_UpdateConfig_ClearsPrevious(t *testing.T) {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     zap.NewNop(),
	}

	// Set initial config
	m.UpdateConfig([]string{"sensor1", "sensor2", "sensor3"})
	if len(m.enabledSet) != 3 {
		t.Fatalf("expected 3 enabled sensors, got %d", len(m.enabledSet))
	}

	// Update with new config - should replace, not append
	m.UpdateConfig([]string{"sensor4"})
	if len(m.enabledSet) != 1 {
		t.Errorf("expected 1 enabled sensor after update, got %d", len(m.enabledSet))
	}
	if !m.enabledSet["sensor4"] {
		t.Error("expected sensor4 to be enabled")
	}
	if m.enabledSet["sensor1"] {
		t.Error("expected sensor1 to be disabled after update")
	}
}

func TestManager_UpdateConfig_EmptyList(t *testing.T) {
	m := &Manager{
		enabledSet: make(map[string]bool),
		logger:     zap.NewNop(),
	}

	// Set initial config
	m.UpdateConfig([]string{"sensor1", "sensor2"})

	// Update with empty list
	m.UpdateConfig([]string{})
	if len(m.enabledSet) != 0 {
		t.Errorf("expected 0 enabled sensors after empty update, got %d", len(m.enabledSet))
	}
}
