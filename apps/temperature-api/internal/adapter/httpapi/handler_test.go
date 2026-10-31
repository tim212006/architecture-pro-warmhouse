package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"temperatureapi/internal/domain"
)

type fakeUseCase struct {
	reading  domain.Reading
	err      error
	location string
	sensorID string
}

func (f *fakeUseCase) Execute(_ context.Context, location, sensorID string) (domain.Reading, error) {
	f.location = location
	f.sensorID = sensorID
	return f.reading, f.err
}

func newTestServer(t *testing.T, useCase TemperatureUseCase) *httptest.Server {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(NewHandler(useCase, logger).Routes())
	t.Cleanup(server.Close)
	return server
}

func TestHandlerHealth(t *testing.T) {
	t.Parallel()

	server := newTestServer(t, &fakeUseCase{})
	response, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response error = %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want ok", body.Status)
	}
}

func TestHandlerTemperatureByLocation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 31, 10, 30, 0, 0, time.UTC)
	useCase := &fakeUseCase{
		reading: domain.Reading{
			Value:       22.5,
			Unit:        "celsius",
			Timestamp:   now,
			Location:    "Living Room",
			Status:      "active",
			SensorID:    "1",
			SensorType:  "temperature",
			Description: "Simulated temperature for Living Room",
		},
	}
	server := newTestServer(t, useCase)

	response, err := http.Get(server.URL + "/temperature?location=Living%20Room")
	if err != nil {
		t.Fatalf("GET /temperature error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if useCase.location != "Living Room" {
		t.Fatalf("location = %q, want Living Room", useCase.location)
	}

	var body readingResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response error = %v", err)
	}
	if body.Value != 22.5 || body.SensorID != "1" {
		t.Fatalf("response = %+v", body)
	}
	if body.Timestamp != now.Format(time.RFC3339) {
		t.Fatalf("timestamp = %q, want %q", body.Timestamp, now.Format(time.RFC3339))
	}
}

func TestHandlerTemperatureBySensorID(t *testing.T) {
	t.Parallel()

	useCase := &fakeUseCase{reading: domain.Reading{SensorID: "2"}}
	server := newTestServer(t, useCase)

	response, err := http.Get(server.URL + "/temperature/2")
	if err != nil {
		t.Fatalf("GET /temperature/2 error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if useCase.sensorID != "2" {
		t.Fatalf("sensor id = %q, want 2", useCase.sensorID)
	}
}

func TestHandlerErrors(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		server := newTestServer(t, &fakeUseCase{})
		response, err := http.Get(server.URL + "/unknown")
		if err != nil {
			t.Fatalf("GET /unknown error = %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		t.Parallel()

		server := newTestServer(t, &fakeUseCase{})
		response, err := http.Post(server.URL+"/temperature", "application/json", nil)
		if err != nil {
			t.Fatalf("POST /temperature error = %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusMethodNotAllowed)
		}
	})

	t.Run("use case error", func(t *testing.T) {
		t.Parallel()

		server := newTestServer(t, &fakeUseCase{err: errors.New("boom")})
		response, err := http.Get(server.URL + "/temperature")
		if err != nil {
			t.Fatalf("GET /temperature error = %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
		}
	})
}
