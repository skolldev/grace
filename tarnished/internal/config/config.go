package config

import (
	"fmt"
	"net/url"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

const (
	MinInterval     = 1 * time.Second
	MaxInterval     = 1 * time.Hour
	DefaultInterval = 10 * time.Second
)

type Config struct {
	Server         string        `mapstructure:"server"`
	APIKey         string        `mapstructure:"api_key"`
	Interval       time.Duration `mapstructure:"interval"`
	LogLevel       string        `mapstructure:"log_level"`
	SensorsEnabled bool          `mapstructure:"sensors_enabled"`
}

func DefaultConfigPath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\Grace\agent.yaml`
	}
	return "/etc/grace/agent.yaml"
}

func DefaultStatePath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\Tarnished\state.json`
	}
	return "/var/lib/tarnished/state.json"
}

func Load(cfgFile string) (*Config, error) {
	// Set defaults
	viper.SetDefault("interval", DefaultInterval)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("sensors_enabled", false)

	// Bind environment variables
	viper.SetEnvPrefix("GRACE")
	viper.BindEnv("server", "GRACE_SERVER")
	viper.BindEnv("api_key", "GRACE_API_KEY")
	viper.BindEnv("interval", "GRACE_INTERVAL")
	viper.BindEnv("log_level", "GRACE_LOG_LEVEL")
	viper.BindEnv("sensors_enabled", "GRACE_SENSORS_ENABLED")

	// Try to read config file (optional)
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigFile(DefaultConfigPath())
	}
	viper.SetConfigType("yaml")
	_ = viper.ReadInConfig() // Ignore error if file doesn't exist

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func Validate(cfg *Config) error {
	if cfg.Server == "" {
		return fmt.Errorf("server URL is required (--server or GRACE_SERVER)")
	}

	// Validate URL format
	parsedURL, err := url.Parse(cfg.Server)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("server URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("server URL must include a host")
	}

	// Validate interval bounds
	if cfg.Interval < MinInterval {
		return fmt.Errorf("interval must be at least %v, got: %v", MinInterval, cfg.Interval)
	}
	if cfg.Interval > MaxInterval {
		return fmt.Errorf("interval must be at most %v, got: %v", MaxInterval, cfg.Interval)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if cfg.LogLevel != "" && !validLogLevels[cfg.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", cfg.LogLevel)
	}

	return nil
}
