package usecase

import (
	"context"
	"math"
	"testing"
	"time"

	"temperatureapi/internal/domain"
)

type fixedGenerator struct {
	value float64
}

func (g fixedGenerator) Float64(_, _ float64) float64 {
	return g.value
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestGetTemperatureExecute(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 31, 10, 30, 0, 0, time.UTC)
	useCase := NewGetTemperature(
		domain.NewRoomDirectory(),
		fixedGenerator{value: 22.55},
		fixedClock{now: now},
	)

	reading, err := useCase.Execute(context.Background(), "Living Room", "")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if math.Abs(reading.Value-22.6) > 1e-9 {
		t.Fatalf("value = %v, want 22.6", reading.Value)
	}
	if reading.Unit != "celsius" {
		t.Fatalf("unit = %q, want celsius", reading.Unit)
	}
	if reading.Location != "Living Room" {
		t.Fatalf("location = %q, want Living Room", reading.Location)
	}
	if reading.SensorID != "1" {
		t.Fatalf("sensor id = %q, want 1", reading.SensorID)
	}
	if reading.Status != "active" {
		t.Fatalf("status = %q, want active", reading.Status)
	}
	if reading.SensorType != "temperature" {
		t.Fatalf("sensor type = %q, want temperature", reading.SensorType)
	}
	if !reading.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %v, want %v", reading.Timestamp, now)
	}
	if reading.Description != "Simulated temperature for Living Room" {
		t.Fatalf("description = %q", reading.Description)
	}
}
