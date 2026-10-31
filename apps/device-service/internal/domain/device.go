package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrInvalidInput   = errors.New("invalid input")
)

type Device struct {
	ID            int64          `json:"id"`
	ExternalID    string         `json:"external_id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Location      string         `json:"location"`
	Unit          string         `json:"unit"`
	Status        string         `json:"status"`
	DesiredState  map[string]any `json:"desired_state"`
	ReportedState map[string]any `json:"reported_state"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type DeviceInput struct {
	ExternalID string         `json:"external_id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location"`
	Unit       string         `json:"unit"`
	Status     string         `json:"status"`
	State      map[string]any `json:"state"`
}

func (i *DeviceInput) Normalize() {
	i.ExternalID = strings.TrimSpace(i.ExternalID)
	i.Name = strings.TrimSpace(i.Name)
	i.Type = strings.TrimSpace(i.Type)
	i.Location = strings.TrimSpace(i.Location)
	i.Unit = strings.TrimSpace(i.Unit)
	i.Status = strings.TrimSpace(i.Status)

	if i.Status == "" {
		i.Status = "offline"
	}
	if i.State == nil {
		i.State = map[string]any{}
	}
}

func (i DeviceInput) Validate() error {
	if i.ExternalID == "" {
		return fmt.Errorf("%w: external_id is required", ErrInvalidInput)
	}
	if i.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if i.Type == "" {
		return fmt.Errorf("%w: type is required", ErrInvalidInput)
	}
	if i.Location == "" {
		return fmt.Errorf("%w: location is required", ErrInvalidInput)
	}
	return nil
}

type StateInput struct {
	Status string         `json:"status"`
	State  map[string]any `json:"state"`
}

func (i *StateInput) Normalize() {
	i.Status = strings.TrimSpace(i.Status)
	if i.State == nil {
		i.State = map[string]any{}
	}
}
