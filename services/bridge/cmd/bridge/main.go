package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/tennex/bridge/internal/config"
	"github.com/tennex/bridge/internal/logging"
	"github.com/tennex/bridge/internal/server"
)

// Version information (set via build flags)
var (
	Version   = "0.1.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override version from build-time information
	cfg.AppVersion = Version

	// Initialize logger
	logger, err := logging.NewLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create service logger with context
	serviceLogger := logging.NewContextLogger(logger, cfg.AppName, cfg.AppVersion)

	// Log startup information
	serviceLogger.Info("Starting Tennex Bridge Service",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("git_commit", GitCommit))

	// Log configuration (with sensitive data masked)
	cfg.LogConfig(serviceLogger)

	// Run the application
	if err := run(context.Background(), cfg, serviceLogger); err != nil {
		serviceLogger.Fatal("Application failed", zap.Error(err))
	}

	serviceLogger.Info("Bridge service stopped")
}

func run(ctx context.Context, cfg *config.Config, logger *zap.Logger) error {
	// Create a context that cancels on interrupt signals
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Initializing bridge components (minimal mode)")

	// Initialize HTTP server
	httpLogger := logging.HTTPLogger(logger)
	httpServer := server.New(server.Config{
		Port:          cfg.HTTPPort,
		Logger:        httpLogger,
		DatabaseURL:   cfg.DatabaseURL,
		EnablePprof:   cfg.Dev.EnablePprof,
		EnableMetrics: cfg.Dev.EnableMetrics,
	})

	// Start HTTP server
	if err := httpServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	endpoints := []string{"/health", "/ready", "/stats", "/connect-minimal", "/debug/config", "/debug/whatsapp"}
	logger.Info("HTTP server started",
		zap.Int("port", cfg.HTTPPort),
		zap.Strings("endpoints", endpoints))

	logger.Info("Bridge service fully initialized and running (multi-tenant mode)")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	logger.Info("Shutting down bridge service...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping HTTP server", zap.Error(err))
	}

	logger.Info("Graceful shutdown completed")
	return nil
}
