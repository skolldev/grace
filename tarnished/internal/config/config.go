package config

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   string        `mapstructure:"server"`
	Token    string        `mapstructure:"token"`
	Interval time.Duration `mapstructure:"interval"`
	LogLevel string        `mapstructure:"log_level"`
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
	viper.SetDefault("interval", 10*time.Second)
	viper.SetDefault("log_level", "info")

	// Bind environment variables
	viper.SetEnvPrefix("GRACE")
	viper.BindEnv("server", "GRACE_SERVER")
	viper.BindEnv("token", "GRACE_TOKEN")
	viper.BindEnv("interval", "GRACE_INTERVAL")
	viper.BindEnv("log_level", "GRACE_LOG_LEVEL")

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
	return nil
}
