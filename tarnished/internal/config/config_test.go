package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
}

func TestLoad_Defaults(t *testing.T) {
	resetViper()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Interval != DefaultInterval {
		t.Errorf("Interval = %v, want %v", cfg.Interval, DefaultInterval)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %s, want info", cfg.LogLevel)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	resetViper()

	// Set environment variables
	os.Setenv("GRACE_SERVER", "http://env-server.local")
	os.Setenv("GRACE_TOKEN", "env-token-123")
	os.Setenv("GRACE_INTERVAL", "30s")
	os.Setenv("GRACE_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("GRACE_SERVER")
		os.Unsetenv("GRACE_TOKEN")
		os.Unsetenv("GRACE_INTERVAL")
		os.Unsetenv("GRACE_LOG_LEVEL")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server != "http://env-server.local" {
		t.Errorf("Server = %s, want http://env-server.local", cfg.Server)
	}
	if cfg.Token != "env-token-123" {
		t.Errorf("Token = %s, want env-token-123", cfg.Token)
	}
	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %s, want debug", cfg.LogLevel)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "agent.yaml")

	content := `server: http://file-server.local
token: file-token-456
interval: 1m
log_level: warn`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server != "http://file-server.local" {
		t.Errorf("Server = %s, want http://file-server.local", cfg.Server)
	}
	if cfg.Token != "file-token-456" {
		t.Errorf("Token = %s, want file-token-456", cfg.Token)
	}
	if cfg.Interval != time.Minute {
		t.Errorf("Interval = %v, want 1m", cfg.Interval)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %s, want warn", cfg.LogLevel)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "agent.yaml")

	content := `server: http://file-server.local
token: file-token`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("GRACE_SERVER", "http://env-override.local")
	defer os.Unsetenv("GRACE_SERVER")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Env should override file
	if cfg.Server != "http://env-override.local" {
		t.Errorf("Server = %s, want http://env-override.local", cfg.Server)
	}
}

func TestLoad_MissingConfigFileIsOK(t *testing.T) {
	resetViper()

	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load should not fail for missing config: %v", err)
	}

	// Should still have defaults
	if cfg.Interval != DefaultInterval {
		t.Errorf("Interval = %v, want %v", cfg.Interval, DefaultInterval)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server:   "https://example.com",
		Token:    "token",
		Interval: 10 * time.Second,
		LogLevel: "info",
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidate_MissingServer(t *testing.T) {
	cfg := &Config{
		Server:   "",
		Interval: 10 * time.Second,
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for missing server")
	}
}

func TestValidate_InvalidServerURL(t *testing.T) {
	tests := []struct {
		name   string
		server string
	}{
		{"no scheme", "example.com"},
		{"invalid scheme", "ftp://example.com"},
		{"file scheme", "file:///path"},
		{"missing host", "http://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   tt.server,
				Interval: 10 * time.Second,
			}
			err := Validate(cfg)
			if err == nil {
				t.Errorf("expected validation error for %s", tt.server)
			}
		})
	}
}

func TestValidate_ValidServerURLs(t *testing.T) {
	tests := []struct {
		name   string
		server string
	}{
		{"http with port", "http://localhost:8080"},
		{"https", "https://example.com"},
		{"http with path", "http://example.com/api"},
		{"ip address", "http://192.168.1.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   tt.server,
				Interval: 10 * time.Second,
				LogLevel: "info",
			}
			err := Validate(cfg)
			if err != nil {
				t.Errorf("unexpected validation error for %s: %v", tt.server, err)
			}
		})
	}
}

func TestValidate_IntervalBounds(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		wantErr  bool
	}{
		{"negative", -1 * time.Second, true},
		{"zero", 0, true},
		{"below minimum", 500 * time.Millisecond, true},
		{"at minimum", MinInterval, false},
		{"normal", 30 * time.Second, false},
		{"at maximum", MaxInterval, false},
		{"above maximum", 2 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   "http://example.com",
				Interval: tt.interval,
				LogLevel: "info",
			}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr = %v, got err = %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Server:   "http://example.com",
		Interval: 10 * time.Second,
		LogLevel: "verbose",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for invalid log level")
	}
}

func TestValidate_ValidLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", ""}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := &Config{
				Server:   "http://example.com",
				Interval: 10 * time.Second,
				LogLevel: level,
			}
			err := Validate(cfg)
			if err != nil {
				t.Errorf("unexpected error for log level %q: %v", level, err)
			}
		})
	}
}

func TestDefaultPaths(t *testing.T) {
	// Just verify the functions return non-empty strings
	cfgPath := DefaultConfigPath()
	if cfgPath == "" {
		t.Error("DefaultConfigPath returned empty string")
	}

	statePath := DefaultStatePath()
	if statePath == "" {
		t.Error("DefaultStatePath returned empty string")
	}
}

func TestLoad_SensorsEnabledDefault(t *testing.T) {
	resetViper()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.SensorsEnabled {
		t.Error("SensorsEnabled should default to false")
	}
}

func TestLoad_SensorsEnabledEnv(t *testing.T) {
	resetViper()

	os.Setenv("GRACE_SERVER", "http://test.local")
	os.Setenv("GRACE_SENSORS_ENABLED", "true")
	defer func() {
		os.Unsetenv("GRACE_SERVER")
		os.Unsetenv("GRACE_SENSORS_ENABLED")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.SensorsEnabled {
		t.Error("SensorsEnabled should be true when GRACE_SENSORS_ENABLED=true")
	}
}
