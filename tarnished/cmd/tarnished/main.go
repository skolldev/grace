package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grace/tarnished/internal/agent"
	"github.com/grace/tarnished/internal/config"
	"github.com/grace/tarnished/internal/version"
)

var (
	cfgFile  string
	logger   *zap.Logger
	logLevel string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tarnished",
		Short: "System monitoring agent for Grace",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initLogger()
		},
	}

	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().String("server", "", "Erdtree server URL (required)")
	rootCmd.PersistentFlags().String("token", "", "Registration token (required for first run)")
	rootCmd.PersistentFlags().Duration("interval", 10*time.Second, "Metric collection interval")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")

	// Bind to viper
	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Add subcommands
	rootCmd.AddCommand(
		installCmd(),
		uninstallCmd(),
		startCmd(),
		stopCmd(),
		statusCmd(),
		runCmd(),
		versionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initLogger() error {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		level = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      level == zapcore.DebugLevel,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		return err
	}
	return nil
}

func serviceName() string {
	switch runtime.GOOS {
	case "windows":
		return "TarnishedAgent"
	case "darwin":
		return "com.tarnished.agent"
	default:
		return "tarnished-agent"
	}
}

func getServiceConfig() *service.Config {
	return &service.Config{
		Name:        serviceName(),
		DisplayName: "Tarnished Agent",
		Description: "Grace system monitoring agent",
	}
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run agent in foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			if err := config.Validate(cfg); err != nil {
				return err
			}

			ag, err := agent.New(cfg, logger)
			if err != nil {
				return err
			}

			return ag.RunForeground()
		},
	}
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install as system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}
			if err := config.Validate(cfg); err != nil {
				return err
			}

			svcConfig := getServiceConfig()
			// Pass arguments to service
			svcConfig.Arguments = []string{
				"run",
				"--server", cfg.Server,
			}
			if cfg.Token != "" {
				svcConfig.Arguments = append(svcConfig.Arguments, "--token", cfg.Token)
			}

			ag, err := agent.New(cfg, logger)
			if err != nil {
				return err
			}

			s, err := service.New(ag, svcConfig)
			if err != nil {
				return err
			}

			err = s.Install()
			if err != nil {
				return err
			}

			fmt.Println("Service installed successfully")
			return nil
		},
	}
}

func uninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove system service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(cfgFile)
			ag, _ := agent.New(cfg, logger)

			s, err := service.New(ag, getServiceConfig())
			if err != nil {
				return err
			}

			err = s.Uninstall()
			if err != nil {
				return err
			}

			fmt.Println("Service uninstalled successfully")
			return nil
		},
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(cfgFile)
			ag, _ := agent.New(cfg, logger)

			s, err := service.New(ag, getServiceConfig())
			if err != nil {
				return err
			}

			err = s.Start()
			if err != nil {
				return err
			}

			fmt.Println("Service started successfully")
			return nil
		},
	}
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(cfgFile)
			ag, _ := agent.New(cfg, logger)

			s, err := service.New(ag, getServiceConfig())
			if err != nil {
				return err
			}

			err = s.Stop()
			if err != nil {
				return err
			}

			fmt.Println("Service stopped successfully")
			return nil
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load(cfgFile)
			ag, _ := agent.New(cfg, logger)

			s, err := service.New(ag, getServiceConfig())
			if err != nil {
				return err
			}

			status, err := s.Status()
			if err != nil {
				return err
			}

			switch status {
			case service.StatusRunning:
				fmt.Println("Service is running")
			case service.StatusStopped:
				fmt.Println("Service is stopped")
			default:
				fmt.Println("Service status: unknown")
			}
			return nil
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}
}
