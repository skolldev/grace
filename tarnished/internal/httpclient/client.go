package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/grace/tarnished/internal/collector"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func New(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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
		return nil, err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/devices/register",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed: %s - %s", resp.Status, string(bodyBytes))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
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
		return err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/metrics",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push metrics failed: %s - %s", resp.Status, string(bodyBytes))
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
			zap.Error(err))

		// Exponential backoff: 1s, 2s, 4s, 8s, 16s
		backoff := min(time.Duration(1<<attempt)*time.Second, 30*time.Second)
		time.Sleep(backoff)
	}

	return fmt.Errorf("push metrics failed after %d retries: %w", maxRetries, lastErr)
}
