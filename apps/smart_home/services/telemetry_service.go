package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TelemetryService struct {
	BaseURL    string
	HTTPClient *http.Client
}

type TelemetryRecord struct {
	DeviceID   string    `json:"device_id"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
}

type TelemetryReading struct {
	ID         int64     `json:"id"`
	DeviceID   string    `json:"device_id"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type telemetryListResponse struct {
	Items []TelemetryReading `json:"items"`
	Total int                `json:"total"`
}

func NewTelemetryService(baseURL string) *TelemetryService {
	return &TelemetryService{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *TelemetryService) Record(ctx context.Context, record TelemetryRecord) error {
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode telemetry request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/api/v1/telemetry", bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("create telemetry request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("call telemetry-service: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("telemetry-service returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func (s *TelemetryService) List(ctx context.Context, deviceID string, limit int) ([]TelemetryReading, error) {
	query := url.Values{}
	query.Set("device_id", deviceID)
	query.Set("limit", strconv.Itoa(limit))

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+"/api/v1/telemetry?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create telemetry request: %w", err)
	}

	response, err := s.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call telemetry-service: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("telemetry-service returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	var payload telemetryListResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode telemetry response: %w", err)
	}

	return payload.Items, nil
}
