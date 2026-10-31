package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"temperatureapi/internal/adapter/clock"
	"temperatureapi/internal/adapter/generator"
	"temperatureapi/internal/adapter/httpapi"
	"temperatureapi/internal/config"
	"temperatureapi/internal/domain"
	"temperatureapi/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("temperature-api stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()

	rooms := domain.NewRoomDirectory()
	getTemperature := usecase.NewGetTemperature(
		rooms,
		generator.NewRandom(),
		clock.NewSystem(),
	)
	handler := httpapi.NewHandler(getTemperature, logger)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("temperature-api listening", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutting down temperature-api")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
