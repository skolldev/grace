//go:build !windows

package sensors

import "errors"

// Reader is a stub for non-Windows platforms
type Reader struct{}

// NewReader returns an error on non-Windows platforms
func NewReader() (*Reader, error) {
	return nil, errors.New("HWiNFO sensors are only available on Windows")
}

// Close is a no-op
func (r *Reader) Close() error {
	return nil
}

// Read returns an error on non-Windows platforms
func (r *Reader) Read() ([]SensorEntry, error) {
	return nil, errors.New("HWiNFO sensors are only available on Windows")
}
