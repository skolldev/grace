package collector

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap"
)

func init() {
	// Use a short CPU sample interval for faster tests
	CPUSampleInterval = 10 * time.Millisecond
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func TestCollector_New(t *testing.T) {
	logger := testLogger()
	c := New(logger)

	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestCollector_Collect_ReturnsMetrics(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if metrics == nil {
		t.Fatal("metrics should not be nil")
	}
}

func TestCollector_Collect_CPUMetrics(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// CPU cores should match runtime.NumCPU()
	if metrics.CPU.Cores != runtime.NumCPU() {
		t.Errorf("CPU.Cores = %d, want %d", metrics.CPU.Cores, runtime.NumCPU())
	}

	// CPU percent should be between 0 and 100
	if metrics.CPU.Percent < 0 || metrics.CPU.Percent > 100 {
		t.Errorf("CPU.Percent = %f, should be between 0 and 100", metrics.CPU.Percent)
	}
}

func TestCollector_Collect_RAMMetrics(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Total RAM should be positive
	if metrics.RAM.TotalGB <= 0 {
		t.Errorf("RAM.TotalGB = %f, should be positive", metrics.RAM.TotalGB)
	}

	// Used RAM should be non-negative and <= total
	if metrics.RAM.UsedGB < 0 {
		t.Errorf("RAM.UsedGB = %f, should be non-negative", metrics.RAM.UsedGB)
	}
	if metrics.RAM.UsedGB > metrics.RAM.TotalGB {
		t.Errorf("RAM.UsedGB = %f > TotalGB = %f", metrics.RAM.UsedGB, metrics.RAM.TotalGB)
	}

	// RAM percent should be between 0 and 100
	if metrics.RAM.Percent < 0 || metrics.RAM.Percent > 100 {
		t.Errorf("RAM.Percent = %f, should be between 0 and 100", metrics.RAM.Percent)
	}
}

func TestCollector_Collect_UptimeMetrics(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Uptime should be positive (system has been running for at least some time)
	if metrics.UptimeSeconds == 0 {
		t.Error("UptimeSeconds should be positive")
	}
}

func TestCollector_Collect_DiskMetrics(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Should have at least one disk on most systems
	if len(metrics.Disk) == 0 {
		t.Log("Warning: no disk metrics collected (may be expected in some environments)")
		return
	}

	for i, disk := range metrics.Disk {
		// Mount point should not be empty
		if disk.Mount == "" {
			t.Errorf("Disk[%d].Mount is empty", i)
		}

		// Total should be positive
		if disk.TotalGB <= 0 {
			t.Errorf("Disk[%d].TotalGB = %f, should be positive", i, disk.TotalGB)
		}

		// Used should be non-negative
		if disk.UsedGB < 0 {
			t.Errorf("Disk[%d].UsedGB = %f, should be non-negative", i, disk.UsedGB)
		}

		// Percent should be between 0 and 100
		if disk.Percent < 0 || disk.Percent > 100 {
			t.Errorf("Disk[%d].Percent = %f, should be between 0 and 100", i, disk.Percent)
		}
	}
}

func TestCollector_Collect_DiskExcludesVirtualFS(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Check that no virtual filesystems are included
	virtualMounts := []string{"/proc", "/sys", "/dev", "/run"}

	for _, disk := range metrics.Disk {
		for _, vm := range virtualMounts {
			if disk.Mount == vm {
				t.Errorf("Virtual filesystem %s should be excluded", vm)
			}
		}
	}
}

func TestCollector_Collect_NetworkMetrics(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Network metrics should be collected (values depend on network activity)
	// Just verify the fields exist and don't overflow
	if metrics.Network.RxBytes < 0 || metrics.Network.TxBytes < 0 {
		t.Error("Network bytes should not be negative")
	}
}

func TestCollector_Collect_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Collect panicked: %v", r)
		}
	}()

	c := New(testLogger())

	// Run multiple collections to increase chance of catching issues
	for i := 0; i < 3; i++ {
		_, _ = c.Collect()
	}
}

func TestIsVirtualFS(t *testing.T) {
	tests := []struct {
		name       string
		fstype     string
		mountpoint string
		want       bool
	}{
		{"tmpfs type", "tmpfs", "/tmp", true},
		{"proc type", "proc", "/proc", true},
		{"sysfs type", "sysfs", "/sys", true},
		{"devtmpfs type", "devtmpfs", "/dev", true},
		{"cgroup type", "cgroup", "/sys/fs/cgroup", true},
		{"ext4 root", "ext4", "/", false},
		{"ext4 home", "ext4", "/home", false},
		{"xfs data", "xfs", "/data", false},
		{"ntfs windows", "ntfs", "C:\\", false},
		{"proc mountpoint", "ext4", "/proc/something", true},
		{"sys mountpoint", "ext4", "/sys/something", true},
		{"dev mountpoint", "ext4", "/dev/something", true},
		{"run mountpoint", "ext4", "/run/something", true},
		{"snap mountpoint", "squashfs", "/snap/something", true},
		{"overlay type", "overlay", "/var/lib/docker/overlay2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVirtualFS(tt.fstype, tt.mountpoint)
			if got != tt.want {
				t.Errorf("isVirtualFS(%q, %q) = %v, want %v", tt.fstype, tt.mountpoint, got, tt.want)
			}
		})
	}
}

func TestVirtualFSTypes_Coverage(t *testing.T) {
	// Ensure common virtual FS types are covered
	expectedTypes := []string{
		"tmpfs", "devtmpfs", "proc", "sysfs", "devfs",
		"debugfs", "securityfs", "cgroup", "cgroup2",
	}

	for _, fstype := range expectedTypes {
		if !virtualFSTypes[fstype] {
			t.Errorf("virtual FS type %q should be in virtualFSTypes map", fstype)
		}
	}
}

func TestMetrics_JSONSerializable(t *testing.T) {
	c := New(testLogger())
	metrics, err := c.Collect()

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Verify metrics can be serialized (used by HTTP client)
	// This tests that all fields have proper JSON tags
	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal metrics: %v", err)
	}

	var decoded Metrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal metrics: %v", err)
	}

	if decoded.CPU.Cores != metrics.CPU.Cores {
		t.Error("CPU.Cores not preserved through JSON round-trip")
	}
}
