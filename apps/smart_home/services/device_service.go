package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DeviceService struct {
	BaseURL    string
	HTTPClient *http.Client
}

type DevicePayload struct {
	ExternalID string         `json:"external_id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location"`
	Unit       string         `json:"unit"`
	Status     string         `json:"status"`
	State      map[string]any `json:"state"`
}

type DeviceStatePayload struct {
	Status string         `json:"status"`
	State  map[string]any `json:"state"`
}

func NewDeviceService(baseURL string) *DeviceService {
	return &DeviceService{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *DeviceService) Upsert(ctx context.Context, payload DevicePayload) error {
	return s.send(ctx, http.MethodPost, "/api/v1/devices", payload, http.StatusCreated, http.StatusOK)
}

func (s *DeviceService) UpdateState(ctx context.Context, externalID string, payload DeviceStatePayload) error {
	path := "/api/v1/devices/" + url.PathEscape(externalID) + "/state"
	return s.send(ctx, http.MethodPatch, path, payload, http.StatusOK)
}

func (s *DeviceService) Delete(ctx context.Context, externalID string) error {
	path := "/api/v1/devices/" + url.PathEscape(externalID)
	return s.send(ctx, http.MethodDelete, path, nil, http.StatusNoContent, http.StatusNotFound)
}

func (s *DeviceService) send(ctx context.Context, method string, path string, payload any, expectedStatuses ...int) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode device request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, s.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("create device request: %w", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := s.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("call device-service: %w", err)
	}
	defer response.Body.Close()

	for _, status := range expectedStatuses {
		if response.StatusCode == status {
			return nil
		}
	}

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	return fmt.Errorf("device-service returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
}
