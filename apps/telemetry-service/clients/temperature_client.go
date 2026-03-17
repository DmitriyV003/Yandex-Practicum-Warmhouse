package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TemperatureResponse struct {
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Timestamp  time.Time `json:"timestamp"`
	Location   string    `json:"location"`
	Status     string    `json:"status"`
	SensorID   string    `json:"sensor_id"`
	SensorType string    `json:"sensor_type"`
}

type TemperatureClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewTemperatureClient(baseURL string) *TemperatureClient {
	return &TemperatureClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *TemperatureClient) GetByID(sensorID string) (*TemperatureResponse, error) {
	url := fmt.Sprintf("%s/temperature/%s", c.BaseURL, sensorID)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch temperature: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var r TemperatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode temperature: %w", err)
	}
	return &r, nil
}

func (c *TemperatureClient) GetByLocation(location string) (*TemperatureResponse, error) {
	url := fmt.Sprintf("%s/temperature?location=%s", c.BaseURL, location)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch temperature: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var r TemperatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode temperature: %w", err)
	}
	return &r, nil
}
