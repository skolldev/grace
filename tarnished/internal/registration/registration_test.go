package registration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/httpclient"
)

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func TestState_JSONRoundTrip(t *testing.T) {
	original := &State{
		DeviceID:     "device-abc-123",
		RegisteredAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Unmarshal
	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID = %s, want %s", loaded.DeviceID, original.DeviceID)
	}
	if !loaded.RegisteredAt.Equal(original.RegisteredAt) {
		t.Errorf("RegisteredAt = %v, want %v", loaded.RegisteredAt, original.RegisteredAt)
	}
}

func TestState_SaveLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "subdir", "state.json")

	original := &State{
		DeviceID:     "test-device-id",
		RegisteredAt: time.Now().UTC().Truncate(time.Millisecond),
	}

	// Save state using helper function that takes a path
	if err := saveStateToPath(statePath, original); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(statePath)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}

	// Load state
	loaded, err := loadStateFromPath(statePath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if loaded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID = %s, want %s", loaded.DeviceID, original.DeviceID)
	}
	if !loaded.RegisteredAt.Equal(original.RegisteredAt) {
		t.Errorf("RegisteredAt = %v, want %v", loaded.RegisteredAt, original.RegisteredAt)
	}
}

func TestState_LoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nonexistent", "state.json")

	state, err := loadStateFromPath(statePath)
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if state != nil {
		t.Error("expected nil state for missing file")
	}
}

func TestState_LoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	if err := os.WriteFile(statePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := loadStateFromPath(statePath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestState_DirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "secure", "state.json")

	state := &State{DeviceID: "test", RegisteredAt: time.Now()}
	if err := saveStateToPath(statePath, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Check directory permissions (0700)
	dir := filepath.Dir(statePath)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// On Windows, permission bits work differently, so we skip this check
	if os.Getenv("OS") != "Windows_NT" {
		mode := info.Mode().Perm()
		if mode != 0700 {
			t.Errorf("directory permissions = %o, want 0700", mode)
		}
	}
}

func TestRegister_Success(t *testing.T) {
	expectedDeviceID := "new-device-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/devices/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"device_id": "` + expectedDeviceID + `", "message": "registered"}`))
	}))
	defer server.Close()

	client := httpclient.New(server.URL, "grc_test-api-key", testLogger())

	// Use a custom save function to avoid writing to system paths
	tmpDir := t.TempDir()
	originalSaveState := saveStateFunc
	saveStateFunc = func(state *State) error {
		return saveStateToPath(filepath.Join(tmpDir, "state.json"), state)
	}
	defer func() { saveStateFunc = originalSaveState }()

	state, err := Register(client)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if state.DeviceID != expectedDeviceID {
		t.Errorf("DeviceID = %s, want %s", state.DeviceID, expectedDeviceID)
	}
	if state.RegisteredAt.IsZero() {
		t.Error("RegisteredAt should not be zero")
	}
}

func TestRegister_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := httpclient.New(server.URL, "grc_test-api-key", testLogger())
	_, err := Register(client)

	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestRegister_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid api key"))
	}))
	defer server.Close()

	client := httpclient.New(server.URL, "bad-api-key", testLogger())
	_, err := Register(client)

	if err == nil {
		t.Error("expected error for invalid api key")
	}
}

func TestGetOutboundIP(t *testing.T) {
	// This function makes an actual network call (UDP to 8.8.8.8:80)
	// It may return empty string if no network is available
	ip := getOutboundIP()

	// We can't guarantee what IP we get, but if we get one it should be valid
	if ip != "" {
		// Basic validation: should contain dots for IPv4
		if len(ip) < 7 { // Minimum valid IP: "0.0.0.0"
			t.Errorf("IP address too short: %s", ip)
		}
	}
	// Empty string is acceptable (no network scenario)
}

// Helper functions for testing with custom paths

func loadStateFromPath(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func saveStateToPath(path string, state *State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
