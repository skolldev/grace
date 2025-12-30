package sensors

// SensorEntry represents a single sensor reading from HWiNFO
type SensorEntry struct {
	ID         string  // hwinfo:type:sensor:entry
	Name       string  // Original name from HWiNFO
	SensorType string  // temperature, voltage, fan, etc.
	Unit       string  // C, V, RPM, etc.
	Value      float64 // Current reading
	Source     string  // "hwinfo"
}
