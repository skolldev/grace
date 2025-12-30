package collector

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

type Metrics struct {
	CPU           CPUMetrics     `json:"cpu"`
	RAM           RAMMetrics     `json:"ram"`
	Disk          []DiskMetrics  `json:"disk"`
	Network       NetworkMetrics `json:"network"`
	UptimeSeconds uint64         `json:"uptime_seconds"`
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
	cpuPercent, err := cpu.Percent(time.Second, false)
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

	// Disk
	partitions, err := disk.Partitions(false)
	if err != nil {
		c.logger.Warn("failed to get disk partitions", zap.Error(err))
	} else {
		for _, p := range partitions {
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
