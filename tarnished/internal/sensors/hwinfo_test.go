//go:build windows

package sensors

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CPU Package", "cpu_package"},
		{"GPU Core #1", "gpu_core_1"},
		{"VCore", "vcore"},
		{"Fan [Sys]", "fan_sys"},
		{"Temperature (C)", "temperature_c"},
		{"", ""},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBuildSensorID(t *testing.T) {
	tests := []struct {
		sensorType string
		sensorName string
		entryName  string
		expected   string
	}{
		{"temperature", "CPU", "Package", "hwinfo:temperature:cpu:package"},
		{"fan", "System", "Fan 1", "hwinfo:fan:system:fan_1"},
		{"voltage", "", "VCore", "hwinfo:voltage:vcore"},
		{"temperature", "NVIDIA GeForce RTX 3080", "GPU Temperature", "hwinfo:temperature:nvidia_geforce_rtx_3080:gpu_temperature"},
	}

	for _, tt := range tests {
		result := buildSensorID(tt.sensorType, tt.sensorName, tt.entryName)
		if result != tt.expected {
			t.Errorf("buildSensorID(%q, %q, %q) = %q, want %q",
				tt.sensorType, tt.sensorName, tt.entryName, result, tt.expected)
		}
	}
}

func TestSensorTypeString(t *testing.T) {
	tests := []struct {
		sensorType SensorType
		expected   string
	}{
		{SensorTypeTemperature, "temperature"},
		{SensorTypeVoltage, "voltage"},
		{SensorTypeFan, "fan"},
		{SensorTypeCurrent, "current"},
		{SensorTypePower, "power"},
		{SensorTypeClock, "clock"},
		{SensorTypeUsage, "usage"},
		{SensorTypeOther, "other"},
		{SensorType(99), "other"},
	}

	for _, tt := range tests {
		result := tt.sensorType.String()
		if result != tt.expected {
			t.Errorf("SensorType(%d).String() = %q, want %q", tt.sensorType, result, tt.expected)
		}
	}
}
