package collector

import (
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// CPUSampleInterval is the duration for CPU sampling.
// Can be overridden in tests for faster execution.
var CPUSampleInterval = time.Second

// virtualFSTypes contains filesystem types to exclude from disk metrics
var virtualFSTypes = map[string]bool{
	"tmpfs":      true,
	"devtmpfs":   true,
	"proc":       true,
	"sysfs":      true,
	"devfs":      true,
	"debugfs":    true,
	"securityfs": true,
	"cgroup":     true,
	"cgroup2":    true,
	"pstore":     true,
	"bpf":        true,
	"tracefs":    true,
	"hugetlbfs":  true,
	"mqueue":     true,
	"fusectl":    true,
	"configfs":   true,
	"efivarfs":   true,
	"autofs":     true,
	"overlay":    true,
	"squashfs":   true,
}

type Metrics struct {
	CPU           CPUMetrics         `json:"cpu"`
	RAM           RAMMetrics         `json:"ram"`
	Disk          []DiskMetrics      `json:"disk"`
	Network       NetworkMetrics     `json:"network"`
	UptimeSeconds uint64             `json:"uptime_seconds"`
	Sensors       map[string]float64 `json:"sensors,omitempty"`
}

type CPUMetrics struct {
	Percent float64 `json:"percent"`
	Cores   int     `json:"cores"`
}

type RAMMetrics struct {
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	Percent float64 `json:"percent"`
}

type DiskMetrics struct {
	Mount   string  `json:"mount"`
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	Percent float64 `json:"percent"`
}

type NetworkMetrics struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type Collector struct {
	logger *zap.Logger
}

func New(logger *zap.Logger) *Collector {
	return &Collector{logger: logger}
}

func (c *Collector) Collect() (*Metrics, error) {
	m := &Metrics{}

	// CPU - use a short interval for sampling
	cpuPercent, err := cpu.Percent(CPUSampleInterval, false)
	if err != nil {
		c.logger.Warn("failed to collect CPU metrics", zap.Error(err))
	} else if len(cpuPercent) > 0 {
		m.CPU.Percent = cpuPercent[0]
	}
	m.CPU.Cores = runtime.NumCPU()

	// RAM
	vmem, err := mem.VirtualMemory()
	if err != nil {
		c.logger.Warn("failed to collect memory metrics", zap.Error(err))
	} else {
		m.RAM.TotalGB = float64(vmem.Total) / (1024 * 1024 * 1024)
		m.RAM.UsedGB = float64(vmem.Used) / (1024 * 1024 * 1024)
		m.RAM.Percent = vmem.UsedPercent
	}

	// Disk - filter out virtual filesystems
	partitions, err := disk.Partitions(false)
	if err != nil {
		c.logger.Warn("failed to get disk partitions", zap.Error(err))
	} else {
		for _, p := range partitions {
			// Skip virtual filesystems
			if isVirtualFS(p.Fstype, p.Mountpoint) {
				continue
			}

			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				c.logger.Debug("failed to get disk usage", zap.String("mount", p.Mountpoint), zap.Error(err))
				continue
			}
			m.Disk = append(m.Disk, DiskMetrics{
				Mount:   p.Mountpoint,
				TotalGB: float64(usage.Total) / (1024 * 1024 * 1024),
				UsedGB:  float64(usage.Used) / (1024 * 1024 * 1024),
				Percent: usage.UsedPercent,
			})
		}
	}

	// Network - aggregate all interfaces
	netIO, err := net.IOCounters(false)
	if err != nil {
		c.logger.Warn("failed to collect network metrics", zap.Error(err))
	} else if len(netIO) > 0 {
		m.Network.RxBytes = netIO[0].BytesRecv
		m.Network.TxBytes = netIO[0].BytesSent
	}

	// Uptime
	uptime, err := host.Uptime()
	if err != nil {
		c.logger.Warn("failed to get uptime", zap.Error(err))
	} else {
		m.UptimeSeconds = uptime
	}

	return m, nil
}

// isVirtualFS checks if the filesystem should be excluded from disk metrics
func isVirtualFS(fstype, mountpoint string) bool {
	// Check filesystem type
	if virtualFSTypes[strings.ToLower(fstype)] {
		return true
	}

	// Exclude common virtual mount points
	excludePrefixes := []string{"/proc", "/sys", "/dev", "/run", "/snap"}
	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(mountpoint, prefix) {
			return true
		}
	}

	return false
}
