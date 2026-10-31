package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"telemetryservice/internal/domain"
)

type UseCase interface {
	Record(ctx context.Context, input domain.RecordInput) (domain.Reading, error)
	List(ctx context.Context, filter domain.Filter) ([]domain.Reading, error)
	Latest(ctx context.Context, deviceID string, metric string) (domain.Reading, error)
}

type Handler struct {
	useCase UseCase
	logger  *slog.Logger
}

func NewHandler(useCase UseCase, logger *slog.Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/v1/telemetry", h.listReadings)
	mux.HandleFunc("POST /api/v1/telemetry", h.recordReading)
	mux.HandleFunc("GET /api/v1/telemetry/latest", h.latestReading)
	mux.HandleFunc("GET /api/v1/devices/{deviceID}/telemetry", h.listDeviceReadings)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) recordReading(w http.ResponseWriter, r *http.Request) {
	var input domain.RecordInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeBadRequest(w, err)
		return
	}

	reading, err := h.useCase.Record(r.Context(), input)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, reading)
}

func (h *Handler) listReadings(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, r.URL.Query().Get("device_id"))
}

func (h *Handler) listDeviceReadings(w http.ResponseWriter, r *http.Request) {
	h.list(w, r, r.PathValue("deviceID"))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, deviceID string) {
	query := r.URL.Query()

	filter := domain.Filter{
		DeviceID: deviceID,
		Metric:   query.Get("metric"),
	}

	from, err := parseTime(query.Get("from"))
	if err != nil {
		h.writeBadRequest(w, err)
		return
	}
	filter.From = from

	to, err := parseTime(query.Get("to"))
	if err != nil {
		h.writeBadRequest(w, err)
		return
	}
	filter.To = to

	if rawLimit := query.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			h.writeBadRequest(w, errors.New("limit must be a positive integer"))
			return
		}
		filter.Limit = limit
	}

	readings, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, listResponse{
		Items: readings,
		Total: len(readings),
	})
}

func (h *Handler) latestReading(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	deviceID := query.Get("device_id")
	if deviceID == "" {
		h.writeBadRequest(w, errors.New("device_id is required"))
		return
	}

	reading, err := h.useCase.Latest(r.Context(), deviceID, query.Get("metric"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, reading)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.writeBadRequest(w, err)
	case errors.Is(err, domain.ErrReadingNotFound):
		h.writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		h.logger.Error("telemetry-service request failed", "error", err)
		h.writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func (h *Handler) writeBadRequest(w http.ResponseWriter, err error) {
	h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func parseTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, errors.New("from and to must be RFC3339 timestamps")
	}

	parsed = parsed.UTC()
	return &parsed, nil
}

type listResponse struct {
	Items any `json:"items"`
	Total int `json:"total"`
}

type errorResponse struct {
	Error string `json:"error"`
}
