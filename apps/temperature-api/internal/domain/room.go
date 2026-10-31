package domain

import "strings"

const (
	UnknownLocation = "Unknown"
	UnknownSensorID = "0"
)

// Room describes a known room and the temperature range used for simulation.
type Room struct {
	Location       string
	SensorID       string
	MinTemperature float64
	MaxTemperature float64
}

// RoomDirectory resolves a location and sensor ID into a room.
type RoomDirectory struct {
	roomsByLocation map[string]Room
}

// NewRoomDirectory returns the default room catalog.
func NewRoomDirectory() *RoomDirectory {
	rooms := []Room{
		{Location: "Living Room", SensorID: "1", MinTemperature: 20, MaxTemperature: 24},
		{Location: "Bedroom", SensorID: "2", MinTemperature: 18, MaxTemperature: 22},
		{Location: "Kitchen", SensorID: "3", MinTemperature: 21, MaxTemperature: 25},
	}

	roomsByLocation := make(map[string]Room, len(rooms))
	for _, room := range rooms {
		roomsByLocation[room.Location] = room
	}

	return &RoomDirectory{roomsByLocation: roomsByLocation}
}

// Resolve returns the known room for the given location or sensor ID.
// Unknown values fall back to a generic room with a wide temperature range.
func (d *RoomDirectory) Resolve(location, sensorID string) Room {
	location = strings.TrimSpace(location)
	sensorID = strings.TrimSpace(sensorID)

	if room, ok := d.roomsByLocation[location]; ok {
		if sensorID == "" {
			sensorID = room.SensorID
		}
		room.SensorID = sensorID
		return room
	}

	if room, ok := d.roomBySensorID(sensorID); ok {
		return room
	}

	return Room{
		Location:       UnknownLocation,
		SensorID:       UnknownSensorID,
		MinTemperature: 18,
		MaxTemperature: 26,
	}
}

func (d *RoomDirectory) roomBySensorID(sensorID string) (Room, bool) {
	if sensorID == "" {
		return Room{}, false
	}

	for _, room := range d.roomsByLocation {
		if room.SensorID == sensorID {
			return room, true
		}
	}

	return Room{}, false
}
