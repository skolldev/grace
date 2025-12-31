package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/collector"
)

const (
	DefaultTimeout         = 30 * time.Second
	DefaultDialTimeout     = 10 * time.Second
	DefaultKeepAlive       = 30 * time.Second
	DefaultIdleConnTimeout = 90 * time.Second
	DefaultMaxIdleConns    = 10
	MaxBackoffDuration     = 30 * time.Second
)

// BaseBackoffDuration is the base duration for exponential backoff.
// Can be overridden in tests for faster execution.
var BaseBackoffDuration = time.Second

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

func New(baseURL string, apiKey string, logger *zap.Logger) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: DefaultKeepAlive,
		}).DialContext,
		MaxIdleConns:        DefaultMaxIdleConns,
		MaxIdleConnsPerHost: DefaultMaxIdleConns,
		IdleConnTimeout:     DefaultIdleConnTimeout,
		DisableCompression:  false,
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
		},
		logger: logger,
	}
}

// doRequest executes an HTTP request with the API key authorization header.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	return c.httpClient.Do(req)
}

type RegisterRequest struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	IPAddress string `json:"ip_address,omitempty"`
}

type RegisterResponse struct {
	DeviceID string `json:"device_id"`
	Message  string `json:"message"`
}

func (c *Client) Register(req *RegisterRequest) (*RegisterResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal registration request: %w", err)
	}

	httpReq, err := http.NewRequest(
		"POST",
		c.baseURL+"/api/devices/register",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("registration failed (status %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("registration failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse registration response: %w", err)
	}

	return &result, nil
}

type MetricsPayload struct {
	DeviceID  string             `json:"device_id"`
	Timestamp time.Time          `json:"timestamp"`
	Metrics   *collector.Metrics `json:"metrics"`
}

type MetricsResponse struct {
	Status string `json:"status"`
}

func (c *Client) PushMetrics(deviceID string, timestamp time.Time, metrics *collector.Metrics) error {
	payload := MetricsPayload{
		DeviceID:  deviceID,
		Timestamp: timestamp,
		Metrics:   metrics,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics payload: %w", err)
	}

	httpReq, err := http.NewRequest(
		"POST",
		c.baseURL+"/api/metrics",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("push metrics failed (status %d)", resp.StatusCode)
		}
		return fmt.Errorf("push metrics failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) PushMetricsWithRetry(deviceID string, timestamp time.Time, metrics *collector.Metrics, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := c.PushMetrics(deviceID, timestamp, metrics)
		if err == nil {
			return nil
		}

		lastErr = err
		c.logger.Warn("push metrics failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))

		// Exponential backoff: base*1, base*2, base*4, etc. (capped at MaxBackoffDuration)
		backoff := min(time.Duration(1<<attempt)*BaseBackoffDuration, MaxBackoffDuration)
		time.Sleep(backoff)
	}

	return fmt.Errorf("push metrics failed after %d retries: %w", maxRetries, lastErr)
}

// SensorInfo represents a sensor to report to the server
type SensorInfo struct {
	SensorID   string `json:"sensor_id"`
	Name       string `json:"name"`
	SensorType string `json:"sensor_type"`
	Unit       string `json:"unit"`
	Source     string `json:"source"`
}

type ReportSensorsRequest struct {
	Sensors []SensorInfo `json:"sensors"`
}

type ReportSensorsResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type SensorConfigResponse struct {
	Enabled []string `json:"enabled"`
}

// ReportSensors reports available sensors to the server
func (c *Client) ReportSensors(deviceID string, sensors []SensorInfo) error {
	req := ReportSensorsRequest{Sensors: sensors}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal sensors request: %w", err)
	}

	httpReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/devices/%s/sensors", c.baseURL, deviceID),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return fmt.Errorf("failed to report sensors: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("report sensors failed (status %d)", resp.StatusCode)
		}
		return fmt.Errorf("report sensors failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetSensorConfig fetches the sensor configuration from the server
func (c *Client) GetSensorConfig(deviceID string) ([]string, error) {
	httpReq, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/devices/%s/sensors/config", c.baseURL, deviceID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doRequest(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("get sensor config failed (status %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("get sensor config failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var config SensorConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse sensor config: %w", err)
	}

	return config.Enabled, nil
}
