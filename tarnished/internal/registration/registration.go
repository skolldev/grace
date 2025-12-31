package registration

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/grace/tarnished/internal/config"
	"github.com/grace/tarnished/internal/httpclient"
)

type State struct {
	DeviceID     string    `json:"device_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

func LoadState() (*State, error) {
	path := config.DefaultStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state = first run
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func SaveState(state *State) error {
	path := config.DefaultStatePath()

	// Ensure directory exists with restrictive permissions (owner-only)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Register(client *httpclient.Client) (*State, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	ipAddr := getOutboundIP()

	req := &httpclient.RegisterRequest{
		Hostname:  hostname,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		IPAddress: ipAddr,
	}

	resp, err := client.Register(req)
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	state := &State{
		DeviceID:     resp.DeviceID,
		RegisteredAt: time.Now().UTC(),
	}

	if err := SaveState(state); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	return state, nil
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return ""
	}
	return localAddr.IP.String()
}
