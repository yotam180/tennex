package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/tennex/eventstream/internal/stream"
)

type Config struct {
	HTTP struct {
		Port int    `koanf:"port"`
		Host string `koanf:"host"`
	} `koanf:"http"`

	NATS struct {
		URL string `koanf:"url"`
	} `koanf:"nats"`

	Backend struct {
		URL string `koanf:"url"`
	} `koanf:"backend"`

	Log struct {
		Level string `koanf:"level"`
		JSON  bool   `koanf:"json"`
	} `koanf:"log"`
}

func main() {
	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger, err := setupLogger(config.Log.Level, config.Log.JSON)
	if err != nil {
		fmt.Printf("Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Tennex Event Stream",
		zap.String("version", "1.0.0"),
		zap.Int("http_port", config.HTTP.Port))

	// Setup NATS connection
	natsConn, err := setupNATS(config.NATS.URL, logger)
	if err != nil {
		logger.Fatal("Failed to setup NATS", zap.Error(err))
	}
	defer natsConn.Close()

	// Create stream manager
	streamManager := stream.NewManager(natsConn, config.Backend.URL, logger)

	// Setup servers
	var wg sync.WaitGroup

	// HTTP server for WebSocket connections
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpConfig := struct {
			Port int
			Host string
		}{
			Port: config.HTTP.Port,
			Host: config.HTTP.Host,
		}
		if err := runHTTPServer(ctx, httpConfig, streamManager, logger); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received, stopping servers...")
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()
	logger.Info("All servers stopped gracefully")
}

func loadConfig() (*Config, error) {
	k := koanf.New(".")

	// Load defaults
	config := &Config{}
	config.HTTP.Port = 6002
	config.HTTP.Host = "0.0.0.0"
	config.NATS.URL = "nats://localhost:4222"
	config.Backend.URL = "http://localhost:8000"
	config.Log.Level = "info"
	config.Log.JSON = false

	// Load from file if exists
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		// File doesn't exist, that's okay
	}

	// Load from environment (TENNEX_EVENTSTREAM_ prefix)
	// Map env like TENNEX_EVENTSTREAM_NATS_URL -> nats.url
	if err := k.Load(env.Provider("TENNEX_EVENTSTREAM_", ".", func(s string) string {
		key := s[18:] // Remove TENNEX_EVENTSTREAM_ prefix
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, "_", ".")
		return key
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading env config: %w", err)
	}

	// Also support generic TENNEX_ prefix to align with docker-compose
	if err := k.Load(env.Provider("TENNEX_", ".", func(s string) string {
		key := s[7:]
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, "_", ".")
		return key
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading env config: %w", err)
	}

	// Unmarshal into config struct
	if err := k.Unmarshal("", config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return config, nil
}

func setupLogger(level string, jsonFormat bool) (*zap.Logger, error) {
	var config zap.Config
	if jsonFormat {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return config.Build()
}

func setupNATS(url string, logger *zap.Logger) (*nats.Conn, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("NATS connection established", zap.String("url", url))
	return nc, nil
}

func runHTTPServer(ctx context.Context, httpConfig struct {
	Port int
	Host string
}, streamManager *stream.Manager, logger *zap.Logger) error {

	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// CORS
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// WebSocket endpoint
	router.Get("/ws", streamManager.HandleWebSocket)

	// Health check
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"eventstream"}`))
	})

	addr := fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	logger.Info("Starting Event Stream HTTP server", zap.String("addr", addr))

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP server shutdown error: %w", err)
	}

	logger.Info("Event Stream HTTP server stopped")
	return nil
}
