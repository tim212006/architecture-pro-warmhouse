package domain

import "time"

// Reading is a single simulated temperature measurement.
type Reading struct {
	Value       float64
	Unit        string
	Timestamp   time.Time
	Location    string
	Status      string
	SensorID    string
	SensorType  string
	Description string
}
