# Tarnished

Go-based system monitoring agent for the Grace platform. Collects system metrics and reports them to the Erdtree server.

## Core Concepts

**Agent Architecture**: Standalone monitoring agent that collects system metrics at configurable intervals and pushes them to the Erdtree API server.

**Device Registration**: On first run, the agent registers itself with the Erdtree server using an API key and receives a device ID for subsequent metric submissions.

**Offline Queue**: If the server is unreachable, metrics are queued locally and retried on subsequent collection cycles to prevent data loss.

**System Service**: Can run as a background service (systemd on Linux, launchd on macOS, Windows Service) or in foreground mode for development.

**Sensor Support**: Optional hardware sensor monitoring using platform-specific tools (e.g., hwinfo on Windows) for temperature, fan speed, and voltage readings.

## Project Structure

```
cmd/
└── tarnished/
    └── main.go          # CLI entry point, command definitions

internal/
├── agent/               # Core agent orchestration and lifecycle
├── collector/           # System metrics collection (CPU, memory, disk)
├── config/              # Configuration loading and validation
├── httpclient/          # HTTP client for Erdtree API
├── queue/               # Offline metric queue
├── registration/        # Device registration and state persistence
├── sensors/             # Hardware sensor management
└── version/             # Version information
```

## Configuration

Tarnished can be configured via config file, environment variables, or command-line flags.

### Config File Locations

**Linux/macOS**: `/etc/grace/agent.yaml`
**Windows**: `C:\ProgramData\Grace\agent.yaml`

### Config File Format

```yaml
server: "http://localhost:8000"
api_key: "grc_xxxxxxxxxxxx"
interval: 10s
log_level: info
sensors_enabled: false
```

### Environment Variables

All config values can be set via environment variables with `GRACE_` prefix:

```bash
GRACE_SERVER=http://localhost:8000
GRACE_API_KEY=grc_xxxxxxxxxxxx
GRACE_INTERVAL=10s
GRACE_LOG_LEVEL=info
GRACE_SENSORS_ENABLED=false
```

### Command-Line Flags

```bash
--server         Erdtree server URL (required)
--api-key        API key for authentication (required)
--interval       Metric collection interval (default: 10s, min: 1s, max: 1h)
--log-level      Log level: debug, info, warn, error (default: info)
--config         Path to config file
```

### State Persistence

Device registration state is stored after first successful registration:

**Linux/macOS**: `/var/lib/tarnished/state.json`
**Windows**: `C:\ProgramData\Tarnished\state.json`

## Development Setup

### Prerequisites

- Go 1.21 or higher
- make (optional, for build automation)

### Install Dependencies

```bash
cd tarnished
go mod download
```

### Build

```bash
# Build for current platform
make build
# or
go build -o bin/tarnished ./cmd/tarnished

# Cross-compile for all platforms
make build-all
```

Binaries are created in the `bin/` directory.

## Usage

### Running in Foreground (Development)

```bash
./bin/tarnished run --server http://localhost:8000 --api-key grc_your_api_key
```

### Installing as System Service

```bash
# Install (requires config file or flags)
sudo ./bin/tarnished install --server http://localhost:8000 --api-key grc_your_api_key

# Start the service
sudo ./bin/tarnished start

# Check status
./bin/tarnished status

# Stop the service
sudo ./bin/tarnished stop

# Uninstall
sudo ./bin/tarnished uninstall
```

### Service Names

- **Linux**: `tarnished-agent`
- **macOS**: `com.tarnished.agent`
- **Windows**: `TarnishedAgent`

## Testing

```bash
# Run all tests
make test
# or
go test -v ./...

# Run tests for specific package
go test -v ./internal/config
```

## Commands

| Command     | Description                                           |
| ----------- | ----------------------------------------------------- |
| `run`       | Run agent in foreground (blocks until SIGINT/SIGTERM) |
| `install`   | Install agent as system service                       |
| `uninstall` | Remove system service                                 |
| `start`     | Start the service                                     |
| `stop`      | Stop the service                                      |
| `status`    | Show service status                                   |
| `version`   | Print version information                             |

## Metrics Collection

The agent collects the following system metrics:

- **CPU**: Usage percentage, per-core usage
- **Memory**: Total, used, free, usage percentage
- **Disk**: Per-partition usage, read/write stats
- **Network**: Interface statistics
- **System**: Uptime, load averages

When `sensors_enabled: true`, hardware sensor data is also collected:

- **Temperature**: CPU, GPU, motherboard sensors
- **Fan Speed**: All detected fan sensors
- **Voltage**: Power rail readings

## Build Information

Version information is embedded at build time via ldflags in the Makefile:

- **Version**: Git tag or "dev"
- **Commit**: Short commit hash
- **BuildDate**: UTC timestamp

View with `./bin/tarnished version`

## Dependencies

Key dependencies (see `go.mod` for full list):

- **cobra**: Command-line interface framework
- **viper**: Configuration management
- **zap**: Structured logging
- **gopsutil**: Cross-platform system metrics
- **kardianos/service**: System service management
