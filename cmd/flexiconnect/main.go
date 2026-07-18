package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashutosh-ak01/flexiconnect/pkg/config"
	"github.com/ashutosh-ak01/flexiconnect/pkg/integration"
	"github.com/ashutosh-ak01/flexiconnect/pkg/secret"
	"github.com/ashutosh-ak01/flexiconnect/pkg/server"
	"github.com/ashutosh-ak01/flexiconnect/pkg/track"
)

// Injected during build time via -ldflags linker settings
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting FlexiConnect Daemon", "version", Version, "commit", Commit, "build_time", BuildTime)

	// In a real scenario, this PostgresRegistry would connect to a real DB based on env vars
	registry := config.NewInMemoryRegistry()
	secretsProvider := secret.NewEnvSecretProvider()
	auditTracker := track.NewLogTracker()

	engine := integration.NewEngine(registry, secretsProvider, auditTracker)

	srv := server.NewServer(engine)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("Server listening", "port", port)
		serverErrors <- httpServer.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		slog.Info("Initiating graceful shutdown", "signal", sig)

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("Graceful shutdown failed", "error", err)
			if err := httpServer.Close(); err != nil {
				slog.Error("Forced shutdown failed", "error", err)
			}
			os.Exit(1)
		}
		slog.Info("Server stopped cleanly")
	}
}
