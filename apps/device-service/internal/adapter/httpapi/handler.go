package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"deviceservice/internal/domain"
)

type UseCase interface {
	Register(ctx context.Context, input domain.DeviceInput) (domain.Device, error)
	List(ctx context.Context) ([]domain.Device, error)
	Get(ctx context.Context, externalID string) (domain.Device, error)
	UpdateState(ctx context.Context, externalID string, input domain.StateInput) (domain.Device, error)
	Delete(ctx context.Context, externalID string) error
	CreateCommand(ctx context.Context, externalID string, input domain.CommandInput) (domain.Command, error)
	ListCommands(ctx context.Context, externalID string) ([]domain.Command, error)
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
	mux.HandleFunc("GET /api/v1/devices", h.listDevices)
	mux.HandleFunc("POST /api/v1/devices", h.registerDevice)
	mux.HandleFunc("GET /api/v1/devices/{externalID}", h.getDevice)
	mux.HandleFunc("DELETE /api/v1/devices/{externalID}", h.deleteDevice)
	mux.HandleFunc("PATCH /api/v1/devices/{externalID}/state", h.updateState)
	mux.HandleFunc("GET /api/v1/devices/{externalID}/commands", h.listCommands)
	mux.HandleFunc("POST /api/v1/devices/{externalID}/commands", h.createCommand)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.useCase.List(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, listResponse{
		Items: devices,
		Total: len(devices),
	})
}

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	var input domain.DeviceInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeBadRequest(w, err)
		return
	}

	device, err := h.useCase.Register(r.Context(), input)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
	device, err := h.useCase.Get(r.Context(), r.PathValue("externalID"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) updateState(w http.ResponseWriter, r *http.Request) {
	var input domain.StateInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeBadRequest(w, err)
		return
	}

	device, err := h.useCase.UpdateState(r.Context(), r.PathValue("externalID"), input)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, device)
}

func (h *Handler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	if err := h.useCase.Delete(r.Context(), r.PathValue("externalID")); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createCommand(w http.ResponseWriter, r *http.Request) {
	var input domain.CommandInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeBadRequest(w, err)
		return
	}

	command, err := h.useCase.CreateCommand(r.Context(), r.PathValue("externalID"), input)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusAccepted, command)
}

func (h *Handler) listCommands(w http.ResponseWriter, r *http.Request) {
	commands, err := h.useCase.ListCommands(r.Context(), r.PathValue("externalID"))
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, listResponse{
		Items: commands,
		Total: len(commands),
	})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.writeBadRequest(w, err)
	case errors.Is(err, domain.ErrDeviceNotFound):
		h.writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		h.logger.Error("device-service request failed", "error", err)
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

type listResponse struct {
	Items any `json:"items"`
	Total int `json:"total"`
}

type errorResponse struct {
	Error string `json:"error"`
}
