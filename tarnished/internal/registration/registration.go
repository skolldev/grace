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

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Register(client *httpclient.Client, token string) (*State, error) {
	hostname, _ := os.Hostname()
	ipAddr := getOutboundIP()

	req := &httpclient.RegisterRequest{
		Token:     token,
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

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
