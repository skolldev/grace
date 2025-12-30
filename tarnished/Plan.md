# Tarnished Agent Plan

## Overview

A lightweight, cross-platform monitoring agent written in Go. Collects system metrics and reports them to the Erdtree server. Installs itself as a system service.

---

## Core Responsibilities

### 1. Registration

- On first run, agent has no identity
- Requires a registration token (provided via CLI flag or env var)
- Calls `POST /api/devices/register` with:
  - Token
  - Hostname
  - OS (linux/windows/darwin)
  - Architecture (amd64/arm64)
  - IP address (best-effort detection)
- Receives and persists `device_id` locally
- Subsequent runs skip registration, use stored ID

### 2. Metric Collection

Collect the following at configurable intervals (default: 10 seconds):

| Metric                    | Linux                       | Windows  | macOS    |
| ------------------------- | --------------------------- | -------- | -------- |
| CPU %                     | `/proc/stat` or gopsutil    | gopsutil | gopsutil |
| RAM used/total            | `/proc/meminfo` or gopsutil | gopsutil | gopsutil |
| Disk used/total per mount | gopsutil                    | gopsutil | gopsutil |
| Network rx/tx bytes       | `/proc/net/dev` or gopsutil | gopsutil | gopsutil |
| Uptime                    | `/proc/uptime` or gopsutil  | gopsutil | gopsutil |

**Optional / Future:**

- CPU temperature (Linux: lm-sensors / `/sys/class/thermal`, Windows: HWiNFO shared memory)
- GPU usage/temp (nvidia-smi, HWiNFO)
- Top processes by CPU/RAM
- Disk I/O rates
- Service status checks

### 3. Metric Reporting

- Push metrics to `POST /api/metrics` as JSON
- Payload structure:
  ```json
  {
    "device_id": "uuid",
    "timestamp": "2025-01-02T10:00:00Z",
    "metrics": {
      "cpu": { "percent": 45.2, "cores": 8 },
      "ram": { "total_gb": 32, "used_gb": 18.5, "percent": 57.8 },
      "disk": [
        { "mount": "/", "total_gb": 500, "used_gb": 230, "percent": 46.0 }
      ],
      "network": { "rx_bytes": 123456789, "tx_bytes": 987654321 },
      "uptime_seconds": 86400
    }
  }
  ```
- Retry logic on failure (exponential backoff, max 5 retries)
- Queue metrics locally if server unreachable (in-memory, bounded)

### 4. Service Management

Using `kardianos/service`:

| Command           | Action                            |
| ----------------- | --------------------------------- |
| `agent install`   | Install as system service         |
| `agent uninstall` | Remove service                    |
| `agent start`     | Start service                     |
| `agent stop`      | Stop service                      |
| `agent status`    | Check if running                  |
| `agent run`       | Run in foreground (for debugging) |

Service names:

- Linux: `tarnished-agent.service` (systemd)
- Windows: `TarnishedAgent` (Windows Service)
- macOS: `com.tarnished.agent` (launchd)

---

## Configuration

### Sources (priority order)

1. CLI flags
2. Environment variables
3. Config file (`/etc/grace/agent.yaml` or `C:\ProgramData\Grace\agent.yaml`)
4. Defaults

### Options

| Option      | Flag          | Env               | Default     | Description                          |
| ----------- | ------------- | ----------------- | ----------- | ------------------------------------ |
| Server URL  | `--server`    | `GRACE_SERVER`    | -           | Required. Base URL of Erdtree server |
| Token       | `--token`     | `GRACE_TOKEN`     | -           | Registration token (first run only)  |
| Interval    | `--interval`  | `GRACE_INTERVAL`  | `10s`       | Collection interval                  |
| Config path | `--config`    | -                 | OS-specific | Path to config file                  |
| Log level   | `--log-level` | `GRACE_LOG_LEVEL` | `info`      | debug/info/warn/error                |

### Persisted State

Location:

- Linux: `/var/lib/tarnished/state.json`
- Windows: `C:\ProgramData\Tarnished\state.json`
- macOS: `/var/lib/tarnished/state.json`

Contents:

```json
{
  "device_id": "uuid-from-registration",
  "registered_at": "2025-01-02T10:00:00Z"
}
```

---

## CLI Interface

```
tarnished [command] [flags]

Commands:
  install     Install as system service
  uninstall   Remove system service
  start       Start the service
  stop        Stop the service
  status      Show service status
  run         Run in foreground
  version     Print version info

Flags:
  --server string      Erdtree server URL (required)
  --token string       Registration token (required for first run)
  --interval duration  Metric collection interval (default 10s)
  --config string      Config file path
  --log-level string   Log level: debug, info, warn, error (default "info")
```

### Example Usage

```bash
# First-time setup (from install script or manual)
tarnished install --server https://erdtree.example.com --token abc123
tarnished start

# Check status
tarnished status

# Run manually for debugging
tarnished run --server https://erdtree.example.com --log-level debug

# Remove
tarnished stop
tarnished uninstall
```

---

## Error Handling

| Scenario                | Behavior                                              |
| ----------------------- | ----------------------------------------------------- |
| Server unreachable      | Retry with backoff, queue metrics (max 100 in memory) |
| Invalid token           | Log error, exit with code 1                           |
| Registration failed     | Log error, exit with code 1                           |
| Metric collection fails | Log warning, skip that metric type, continue          |
| Config file missing     | Use defaults + CLI flags                              |
| State file missing      | Assume first run, require token                       |

---

## Logging

- Structured logging (JSON in production, pretty in debug)
- Log to stdout when running in foreground
- Log to system journal (Linux) or Event Log (Windows) when running as service
- Log levels: debug, info, warn, error

---

## Dependencies

| Package                         | Purpose                           |
| ------------------------------- | --------------------------------- |
| `github.com/shirou/gopsutil/v3` | Cross-platform system metrics     |
| `github.com/kardianos/service`  | Cross-platform service management |
| `github.com/spf13/cobra`        | CLI framework                     |
| `github.com/spf13/viper`        | Configuration management          |
| `go.uber.org/zap`               | Structured logging                |

---

## Build & Distribution

### Build targets

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o tarnished-linux-amd64
GOOS=linux GOARCH=arm64 go build -o tarnished-linux-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o tarnished-windows-amd64.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o tarnished-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o tarnished-darwin-arm64
```

### Install script

Server serves an install script at `/install.sh` (and `/install.ps1` for Windows).

Linux/macOS:

```bash
curl -sSL https://erdtree.example.com/install.sh | bash -s -- --server https://erdtree.example.com --token abc123
```

Windows (PowerShell):

```powershell
irm https://erdtree.example.com/install.ps1 | iex -Args "--server https://erdtree.example.com --token abc123"
```

Script responsibilities:

1. Detect OS and architecture
2. Download appropriate binary
3. Place in correct location (`/usr/local/bin` or `C:\Program Files\Tarnished`)
4. Run `tarnished install --server X --token Y`
5. Start the service

---

## Future Considerations

- [ ] Agent self-update mechanism
- [ ] Command channel (WebSocket or long-poll) for remote config changes
- [ ] Plugin system for custom metrics
- [ ] Certificate pinning for server communication
- [ ] Compression for metric payloads
- [ ] Local metric buffering to disk (survive agent restarts)
