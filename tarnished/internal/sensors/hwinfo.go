//go:build windows

package sensors

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping = kernel32.NewProc("OpenFileMappingW")
)

const (
	SharedMemPath = "Global\\HWiNFO_SENS_SM2"
	HeaderMagic   = 0x53695748 // 'HWiS'
)

// Header represents the HWiNFO shared memory header (44 bytes)
type Header struct {
	Magic              uint32
	Version            uint32
	Version2           uint32
	LastUpdate         int64
	SensorSectionOff   uint32
	SensorElementSize  uint32
	SensorElementCount uint32
	EntrySectionOff    uint32
	EntryElementSize   uint32
	EntryElementCount  uint32
}

// SensorType represents the type of sensor reading
type SensorType int

const (
	SensorTypeTemperature SensorType = 1
	SensorTypeVoltage     SensorType = 2
	SensorTypeFan         SensorType = 3
	SensorTypeCurrent     SensorType = 4
	SensorTypePower       SensorType = 5
	SensorTypeClock       SensorType = 6
	SensorTypeUsage       SensorType = 7
	SensorTypeOther       SensorType = 8
)

func (t SensorType) String() string {
	switch t {
	case SensorTypeTemperature:
		return "temperature"
	case SensorTypeVoltage:
		return "voltage"
	case SensorTypeFan:
		return "fan"
	case SensorTypeCurrent:
		return "current"
	case SensorTypePower:
		return "power"
	case SensorTypeClock:
		return "clock"
	case SensorTypeUsage:
		return "usage"
	default:
		return "other"
	}
}

type rawSensor struct {
	id       uint32
	instance uint32
	name     string
}

// Reader reads sensors from HWiNFO shared memory
type Reader struct{}

// openFileMapping opens a named file mapping object
func openFileMapping(desiredAccess uint32, inheritHandle bool, name string) (windows.Handle, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}

	inherit := uint32(0)
	if inheritHandle {
		inherit = 1
	}

	ret, _, err := procOpenFileMapping.Call(
		uintptr(desiredAccess),
		uintptr(inherit),
		uintptr(unsafe.Pointer(namePtr)),
	)

	if ret == 0 {
		return 0, err
	}
	return windows.Handle(ret), nil
}

// NewReader creates a new HWiNFO reader
func NewReader() (*Reader, error) {
	// Try to open shared memory to verify HWiNFO is available
	handle, err := openFileMapping(windows.FILE_MAP_READ, false, SharedMemPath)
	if err != nil {
		return nil, fmt.Errorf("HWiNFO not available (is it running with shared memory enabled?): %w", err)
	}
	windows.CloseHandle(handle)

	return &Reader{}, nil
}

// Close releases resources (no-op since we open/close per read)
func (r *Reader) Close() error {
	return nil
}

// Read reads all sensor entries from HWiNFO
func (r *Reader) Read() ([]SensorEntry, error) {
	handle, err := openFileMapping(windows.FILE_MAP_READ, false, SharedMemPath)
	if err != nil {
		return nil, fmt.Errorf("HWiNFO not available: %w", err)
	}
	defer windows.CloseHandle(handle)

	// Map the shared memory
	ptr, err := windows.MapViewOfFile(handle, windows.FILE_MAP_READ, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to map shared memory: %w", err)
	}
	defer windows.UnmapViewOfFile(ptr)

	// Read raw data (1MB should be plenty)
	// Safe: ptr is from MapViewOfFile and kept alive until UnmapViewOfFile
	//nolint:govet // Safe conversion of Windows memory-mapped pointer
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), 1024*1024)

	// Read header manually to avoid Go struct alignment issues
	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != HeaderMagic {
		return nil, fmt.Errorf("invalid HWiNFO magic: got 0x%X, expected 0x%X", magic, HeaderMagic)
	}

	header := Header{
		Magic:              magic,
		Version:            binary.LittleEndian.Uint32(data[4:8]),
		Version2:           binary.LittleEndian.Uint32(data[8:12]),
		LastUpdate:         int64(binary.LittleEndian.Uint64(data[12:20])),
		SensorSectionOff:   binary.LittleEndian.Uint32(data[20:24]),
		SensorElementSize:  binary.LittleEndian.Uint32(data[24:28]),
		SensorElementCount: binary.LittleEndian.Uint32(data[28:32]),
		EntrySectionOff:    binary.LittleEndian.Uint32(data[32:36]),
		EntryElementSize:   binary.LittleEndian.Uint32(data[36:40]),
		EntryElementCount:  binary.LittleEndian.Uint32(data[40:44]),
	}

	// Read sensors (parent devices)
	// Layout: ID(4) + Instance(4) + NameOriginal(128) + NameUser(128) = 264 bytes
	sensors := make([]rawSensor, header.SensorElementCount)
	for i := uint32(0); i < header.SensorElementCount; i++ {
		offset := header.SensorSectionOff + i*header.SensorElementSize
		sensorData := data[offset:]

		nameUser := readCString(sensorData[136:264])
		nameOrig := readCString(sensorData[8:136])
		name := nameUser
		if name == "" {
			name = nameOrig
		}

		sensors[i] = rawSensor{
			id:       binary.LittleEndian.Uint32(sensorData[0:4]),
			instance: binary.LittleEndian.Uint32(sensorData[4:8]),
			name:     name,
		}
	}

	// Read entries (actual sensor readings)
	// Layout: Type(4) + SensorIndex(4) + ID(4) + NameOriginal(128) + NameUser(128) + Unit(16) + Value(8) + ...
	// Offsets: 0-4, 4-8, 8-12, 12-140, 140-268, 268-284, 284-292
	entries := make([]SensorEntry, 0, header.EntryElementCount)
	for i := uint32(0); i < header.EntryElementCount; i++ {
		offset := header.EntrySectionOff + i*header.EntryElementSize
		entryData := data[offset:]

		sensorType := SensorType(binary.LittleEndian.Uint32(entryData[0:4]))
		sensorIndex := binary.LittleEndian.Uint32(entryData[4:8])

		nameOrig := readCString(entryData[12:140])

		unit := readCString(entryData[268:284])
		value := *(*float64)(unsafe.Pointer(&entryData[284]))

		// Build sensor ID: hwinfo:{type}:{sensor_name}:{entry_name}
		sensorName := ""
		if sensorIndex < uint32(len(sensors)) {
			sensorName = sensors[sensorIndex].name
		}

		entryName := sensorName + " " + nameOrig

		sensorID := buildSensorID(sensorType.String(), sensorName, nameOrig)

		entries = append(entries, SensorEntry{
			ID:         sensorID,
			Name:       entryName,
			SensorType: sensorType.String(),
			Unit:       unit,
			Value:      value,
			Source:     "hwinfo",
		})
	}

	return entries, nil
}

func readCString(data []byte) string {
	// Find null terminator
	n := 0
	for n < len(data) && data[n] != 0 {
		n++
	}
	return strings.TrimSpace(string(data[:n]))
}

var sanitizeRegex = regexp.MustCompile(`[^a-z0-9_]`)

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = sanitizeRegex.ReplaceAllString(name, "")
	return name
}

func buildSensorID(sensorType, sensorName, entryName string) string {
	parts := []string{"hwinfo", sensorType}
	if sensorName != "" {
		parts = append(parts, sanitizeName(sensorName))
	}
	parts = append(parts, sanitizeName(entryName))
	return strings.Join(parts, ":")
}
