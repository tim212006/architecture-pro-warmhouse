package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrReadingNotFound = errors.New("reading not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type Reading struct {
	ID         int64     `json:"id"`
	DeviceID   string    `json:"device_id"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type RecordInput struct {
	DeviceID   string     `json:"device_id"`
	Metric     string     `json:"metric"`
	Value      float64    `json:"value"`
	Unit       string     `json:"unit"`
	RecordedAt *time.Time `json:"recorded_at"`
}

func (i *RecordInput) Normalize() {
	i.DeviceID = strings.TrimSpace(i.DeviceID)
	i.Metric = strings.TrimSpace(i.Metric)
	i.Unit = strings.TrimSpace(i.Unit)
	if i.RecordedAt != nil {
		recordedAt := i.RecordedAt.UTC()
		i.RecordedAt = &recordedAt
	}
}

func (i RecordInput) Validate() error {
	if i.DeviceID == "" {
		return fmt.Errorf("%w: device_id is required", ErrInvalidInput)
	}
	if i.Metric == "" {
		return fmt.Errorf("%w: metric is required", ErrInvalidInput)
	}
	return nil
}

type Filter struct {
	DeviceID string
	Metric   string
	From     *time.Time
	To       *time.Time
	Limit    int
}

func (f Filter) Normalize() Filter {
	f.DeviceID = strings.TrimSpace(f.DeviceID)
	f.Metric = strings.TrimSpace(f.Metric)
	if f.Limit <= 0 {
		f.Limit = 100
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	return f
}
