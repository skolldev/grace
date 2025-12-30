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
