package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tennex/bridge/internal/config"
	"github.com/tennex/bridge/internal/events"
	"github.com/tennex/bridge/internal/logging"
	"github.com/tennex/bridge/internal/manager"
	"github.com/tennex/bridge/internal/publisher"
	"github.com/tennex/bridge/internal/server"
	"github.com/tennex/bridge/internal/storage"
	"github.com/tennex/bridge/internal/whatsapp"
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

	// Generate account and device IDs for this instance
	// In a real implementation, these would come from configuration or database
	accountID := cfg.AppName + "-account"
	deviceID := uuid.New().String()

	logger.Info("Initializing bridge components",
		zap.String("account_id", accountID),
		zap.String("device_id", deviceID))

	// Initialize MongoDB storage
	dbLogger := logging.DatabaseLogger(logger)
	mongodb, err := storage.NewMongoDB(ctx, storage.ConnectOptions{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		Logger:   dbLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer mongodb.Close(ctx)

	// Initialize event publisher (console publisher for PoC)
	eventPublisher := publisher.NewConsolePublisher(logging.EventLogger(logger))

	// Initialize event handler
	eventHandler := events.NewHandler(
		logging.EventLogger(logger),
		eventPublisher,
		accountID,
		deviceID,
	)

	// Initialize multi-tenant client manager
	clientManager, err := manager.NewClientManager(manager.ClientManagerConfig{
		Storage:      mongodb,
		Logger:       logging.WhatsAppLogger(logger),
		EventHandler: eventHandler,
		SessionPath:  cfg.WhatsApp.SessionPath,
		DBLogLevel:   cfg.WhatsApp.DBLogLevel,
	})
	if err != nil {
		return fmt.Errorf("failed to create client manager: %w", err)
	}
	defer clientManager.Stop()

	// Optionally initialize legacy single WhatsApp client (for backward compatibility)
	var waClient *whatsapp.Client
	if cfg.Dev.QRInTerminal {
		// Only create legacy client if QR in terminal is enabled (development mode)
		waLogger := logging.WhatsAppLogger(logger)
		waClient, err = whatsapp.NewClient(whatsapp.ClientConfig{
			SessionPath:  cfg.WhatsApp.SessionPath + "/legacy",
			QRInTerminal: cfg.Dev.QRInTerminal,
			DBLogLevel:   cfg.WhatsApp.DBLogLevel,
			HistorySync:  cfg.WhatsApp.HistorySync,
			EventHandler: eventHandler,
			Logger:       waLogger,
		})
		if err != nil {
			logger.Warn("Failed to create legacy WhatsApp client", zap.Error(err))
			waClient = nil
		}
	}

	// Initialize HTTP server
	httpLogger := logging.HTTPLogger(logger)
	httpServer := server.New(server.Config{
		Port:          cfg.HTTPPort,
		Logger:        httpLogger,
		WAClient:      waClient,
		ClientManager: clientManager,
		EnablePprof:   cfg.Dev.EnablePprof,
		EnableMetrics: cfg.Dev.EnableMetrics,
	})

	// Start HTTP server
	if err := httpServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	endpoints := []string{"/health", "/ready", "/stats", "/connect-client", "/debug/config", "/debug/whatsapp"}
	logger.Info("HTTP server started",
		zap.Int("port", cfg.HTTPPort),
		zap.Strings("endpoints", endpoints))

	// Optionally connect legacy WhatsApp client
	if waClient != nil {
		connectCtx, connectCancel := context.WithTimeout(ctx, cfg.WhatsApp.ConnectTimeout)
		defer connectCancel()

		logger.Info("Connecting legacy WhatsApp client...")
		if err := waClient.Connect(connectCtx); err != nil {
			logger.Warn("Failed to connect legacy WhatsApp client", zap.Error(err))
		} else {
			logger.Info("Legacy WhatsApp client connected")
		}
	}

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

	// Disconnect legacy WhatsApp client
	if waClient != nil {
		waClient.Disconnect()
	}

	// Client manager will be stopped by defer

	logger.Info("Graceful shutdown completed")
	return nil
}
