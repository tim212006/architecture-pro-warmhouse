package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"temperatureapi/internal/domain"
)

// TemperatureUseCase returns a simulated temperature reading.
type TemperatureUseCase interface {
	Execute(ctx context.Context, location, sensorID string) (domain.Reading, error)
}

// Handler serves the temperature API over HTTP.
type Handler struct {
	useCase TemperatureUseCase
	logger  *slog.Logger
}

// NewHandler creates an HTTP handler.
func NewHandler(useCase TemperatureUseCase, logger *slog.Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Routes returns the HTTP router for the temperature API.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /temperature", h.getTemperature)
	mux.HandleFunc("GET /temperature/{sensorID}", h.getTemperatureBySensor)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (h *Handler) getTemperature(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	sensorID := query.Get("sensorId")
	if sensorID == "" {
		sensorID = query.Get("sensor_id")
	}
	h.respondWithReading(w, r, query.Get("location"), sensorID)
}

func (h *Handler) getTemperatureBySensor(w http.ResponseWriter, r *http.Request) {
	h.respondWithReading(w, r, "", r.PathValue("sensorID"))
}

func (h *Handler) respondWithReading(w http.ResponseWriter, r *http.Request, location, sensorID string) {
	reading, err := h.useCase.Execute(r.Context(), location, sensorID)
	if err != nil {
		h.logger.Error("failed to generate temperature reading", "error", err)
		h.writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	h.writeJSON(w, http.StatusOK, newReadingResponse(reading))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

type healthResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type readingResponse struct {
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Timestamp   string  `json:"timestamp"`
	Location    string  `json:"location"`
	Status      string  `json:"status"`
	SensorID    string  `json:"sensor_id"`
	SensorType  string  `json:"sensor_type"`
	Description string  `json:"description"`
}

func newReadingResponse(reading domain.Reading) readingResponse {
	return readingResponse{
		Value:       reading.Value,
		Unit:        reading.Unit,
		Timestamp:   reading.Timestamp.Format(time.RFC3339),
		Location:    reading.Location,
		Status:      reading.Status,
		SensorID:    reading.SensorID,
		SensorType:  reading.SensorType,
		Description: reading.Description,
	}
}
