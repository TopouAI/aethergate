package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/topoai/aethergate/apps/api/internal/httpapi"
	"github.com/topoai/aethergate/apps/api/internal/platform"
	postgresstorage "github.com/topoai/aethergate/apps/api/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	repository, repositorySource, closeRepository := configureRepository(logger)
	defer closeRepository()

	server := &http.Server{
		Addr:              envOrDefault("AETHERGATE_API_ADDR", ":8080"),
		Handler:           httpapi.NewHandlerWithRepository(logger, repository, repositorySource),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		logger.Info("aethergate api listening", "address", server.Addr, "repository", repositorySource)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownContext.Done()
	stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("aethergate api stopped")
}

func configureRepository(logger *slog.Logger) (platform.Repository, string, func()) {
	databaseURL := strings.TrimSpace(os.Getenv("AETHERGATE_DATABASE_URL"))
	if databaseURL == "" {
		logger.Warn("using development memory repository", "reason", "AETHERGATE_DATABASE_URL is empty")
		return platform.NewMemoryRepository(), "development-memory", func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	repository, err := postgresstorage.Open(ctx, databaseURL)
	if err != nil {
		logger.Error("postgres repository unavailable", "error", err)
		os.Exit(1)
	}
	if strings.EqualFold(os.Getenv("AETHERGATE_AUTO_MIGRATE"), "true") {
		if err := repository.Migrate(ctx); err != nil {
			repository.Close()
			logger.Error("postgres migration failed", "error", err)
			os.Exit(1)
		}
	}
	return repository, "postgresql", repository.Close
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
