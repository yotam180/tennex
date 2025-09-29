package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/tennex/bridge/db"
	backendGRPC "github.com/tennex/bridge/internal/grpc"
	"github.com/tennex/bridge/internal/handlers"
	"github.com/tennex/bridge/whatsapp"
	"github.com/tennex/shared/auth"
)

const (
	DefaultJWTSecret       = "dev-jwt-secret-change-in-production"
	DefaultPort            = "6003"
	DefaultBackendGRPCAddr = "backend:6001" // Default for Docker, can be overridden
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Shutdown signal received, stopping server...")
		cancel()
	}()

	slog.Info("🚀 Starting Tennex Bridge Service", "version", "1.0.0", "port", DefaultPort)

	// Initialize database storage
	storage, err := db.NewStorage()
	if err != nil {
		slog.Error("Failed to initialize database storage", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Database connection established")

	// Initialize WhatsApp connector (temporarily without backend client)
	var whatsappConnector *whatsapp.WhatsAppConnector

	// Initialize JWT configuration
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = DefaultJWTSecret
		slog.Warn("Using default JWT secret - change for production!")
	}

	// Debug logging for JWT configuration
	slog.Info("🔐 JWT Configuration Debug",
		"secret_length", len(jwtSecret),
		"secret_first_10", jwtSecret[:min(10, len(jwtSecret))],
		"env_jwt_secret_set", os.Getenv("JWT_SECRET") != "",
	)

	jwtConfig := auth.DefaultJWTConfig(jwtSecret)
	slog.Info("✅ JWT authentication configured")

	// Initialize backend gRPC client
	backendAddr := os.Getenv("BACKEND_GRPC_ADDR")
	if backendAddr == "" {
		backendAddr = DefaultBackendGRPCAddr
		slog.Info("Using default backend gRPC address", "addr", backendAddr)
	}

	backendClient, err := backendGRPC.NewBackendClient(backendAddr)
	if err != nil {
		slog.Error("Failed to initialize backend gRPC client", "error", err, "addr", backendAddr)
		os.Exit(1)
	}
	defer backendClient.Close()
	slog.Info("✅ Backend gRPC client connected", "addr", backendAddr)

	// Initialize WhatsApp connector with backend client
	whatsappConnector = whatsapp.NewWhatsAppConnector(storage, backendClient)
	slog.Info("✅ WhatsApp connector initialized")

	// Initialize handlers
	whatsappHandler := handlers.NewWhatsAppHandler(storage, whatsappConnector, backendClient)
	mainHandler := handlers.NewMainHandler(storage, whatsappHandler, jwtConfig)

	// Setup HTTP router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Mount all routes
	r.Mount("/", mainHandler.Routes())

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + DefaultPort,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("🌐 HTTP server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			cancel()
		}
	}()

	slog.Info("✅ Tennex Bridge Service is running!")
	slog.Info("📊 Service endpoints:")
	slog.Info("  Health check: http://localhost:" + DefaultPort + "/health")
	slog.Info("  WhatsApp connect: POST http://localhost:" + DefaultPort + "/whatsapp/connect (requires JWT)")
	slog.Info("  WhatsApp status: GET http://localhost:" + DefaultPort + "/whatsapp/status (requires JWT)")
	slog.Info("  Connections: GET http://localhost:" + DefaultPort + "/connections (requires JWT)")
	slog.Info("🔐 JWT Debug Mode ENABLED - tokens will be logged in detail")

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	slog.Info("🛑 Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Server stopped gracefully")
}
