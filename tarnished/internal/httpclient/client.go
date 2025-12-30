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

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func New(baseURL string, logger *zap.Logger) *Client {
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
		httpClient: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
		},
		logger: logger,
	}
}

type RegisterRequest struct {
	Token     string `json:"token"`
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

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/devices/register",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
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

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/metrics",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
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

		// Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped at MaxBackoffDuration)
		backoff := min(time.Duration(1<<attempt)*time.Second, MaxBackoffDuration)
		time.Sleep(backoff)
	}

	return fmt.Errorf("push metrics failed after %d retries: %w", maxRetries, lastErr)
}
