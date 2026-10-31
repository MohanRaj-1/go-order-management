package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MohanRaj-1/go-order-management/internal/config"
	"github.com/MohanRaj-1/go-order-management/internal/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	appLogger := logger.New(os.Stdout, cfg.LogLevel)

	mux := newRouter()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		appLogger.Info("Starting HTTP server", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	var serverErr error
	serverResultReceived := false

	select {
	case err := <-serverErrors:
		serverResultReceived = true

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr = fmt.Errorf("HTTP server failed: %w", err)
			appLogger.Error("HTTP server failed", "error", err)
		}

	case <-signalCtx.Done():
		appLogger.Info("Shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.HTTPShutdownTimeout,
	)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		appLogger.Error("Graceful shutdown failed", "error", shutdownErr)

		if closeErr := server.Close(); closeErr != nil {
			appLogger.Error("Force closing server failed", "error", closeErr)
			shutdownErr = errors.Join(shutdownErr, closeErr)
		}
	}

	if !serverResultReceived {
		err := <-serverErrors
		serverResultReceived = true

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr = fmt.Errorf("HTTP server failed: %w", err)
			appLogger.Error("HTTP server failed", "error", err)
		}
	}

	appLogger.Info("HTTP server stopped")

	if serverErr != nil && shutdownErr != nil {
		return errors.Join(
			serverErr,
			fmt.Errorf("shutdown failed: %w", shutdownErr),
		)
	}

	if serverErr != nil {
		return serverErr
	}

	if shutdownErr != nil {
		return fmt.Errorf("shutdown failed: %w", shutdownErr)
	}

	return nil
}

func newRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}
