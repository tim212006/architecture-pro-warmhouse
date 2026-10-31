package domain

import "testing"

func TestRoomDirectoryResolve(t *testing.T) {
	t.Parallel()

	directory := NewRoomDirectory()

	tests := []struct {
		name        string
		location    string
		sensorID    string
		wantRoom    string
		wantSensor  string
		wantMinTemp float64
		wantMaxTemp float64
	}{
		{
			name:        "location from query",
			location:    "Living Room",
			wantRoom:    "Living Room",
			wantSensor:  "1",
			wantMinTemp: 20,
			wantMaxTemp: 24,
		},
		{
			name:        "location from sensor id",
			sensorID:    "2",
			wantRoom:    "Bedroom",
			wantSensor:  "2",
			wantMinTemp: 18,
			wantMaxTemp: 22,
		},
		{
			name:        "known location with explicit sensor id",
			location:    "Kitchen",
			sensorID:    "42",
			wantRoom:    "Kitchen",
			wantSensor:  "42",
			wantMinTemp: 21,
			wantMaxTemp: 25,
		},
		{
			name:        "unknown values fall back to defaults",
			wantRoom:    UnknownLocation,
			wantSensor:  UnknownSensorID,
			wantMinTemp: 18,
			wantMaxTemp: 26,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			room := directory.Resolve(test.location, test.sensorID)
			if room.Location != test.wantRoom {
				t.Fatalf("location = %q, want %q", room.Location, test.wantRoom)
			}
			if room.SensorID != test.wantSensor {
				t.Fatalf("sensor id = %q, want %q", room.SensorID, test.wantSensor)
			}
			if room.MinTemperature != test.wantMinTemp || room.MaxTemperature != test.wantMaxTemp {
				t.Fatalf(
					"range = [%v, %v], want [%v, %v]",
					room.MinTemperature,
					room.MaxTemperature,
					test.wantMinTemp,
					test.wantMaxTemp,
				)
			}
		})
	}
}
