package usecase

import (
	"context"
	"math"
	"time"

	"temperatureapi/internal/domain"
)

const (
	celsiusUnit       = "celsius"
	activeStatus      = "active"
	temperatureSensor = "temperature"
)

// RoomDirectory resolves a location and sensor ID into a simulated room.
type RoomDirectory interface {
	Resolve(location, sensorID string) domain.Room
}

// Generator produces a random value within the given range.
type Generator interface {
	Float64(min, max float64) float64
}

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// GetTemperature creates simulated temperature readings.
type GetTemperature struct {
	rooms     RoomDirectory
	generator Generator
	clock     Clock
}

// NewGetTemperature wires the use case dependencies.
func NewGetTemperature(rooms RoomDirectory, generator Generator, clock Clock) *GetTemperature {
	return &GetTemperature{
		rooms:     rooms,
		generator: generator,
		clock:     clock,
	}
}

// Execute returns a simulated reading for the requested location or sensor ID.
func (uc *GetTemperature) Execute(_ context.Context, location, sensorID string) (domain.Reading, error) {
	room := uc.rooms.Resolve(location, sensorID)
	value := uc.generator.Float64(room.MinTemperature, room.MaxTemperature)

	return domain.Reading{
		Value:       math.Round(value*10) / 10,
		Unit:        celsiusUnit,
		Timestamp:   uc.clock.Now().UTC(),
		Location:    room.Location,
		Status:      activeStatus,
		SensorID:    room.SensorID,
		SensorType:  temperatureSensor,
		Description: "Simulated temperature for " + room.Location,
	}, nil
}
