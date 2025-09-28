package main

import (
	"context"
	"fmt"
	"net"
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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/tennex/backend/internal/core"
	"github.com/tennex/backend/internal/grpc/server"
	"github.com/tennex/backend/internal/http/handlers"
	"github.com/tennex/backend/internal/repo"
	dbgen "github.com/tennex/pkg/db/gen"
)

type Config struct {
	HTTP struct {
		Port int    `koanf:"port"`
		Host string `koanf:"host"`
	} `koanf:"http"`

	GRPC struct {
		Port int    `koanf:"port"`
		Host string `koanf:"host"`
	} `koanf:"grpc"`

	Database struct {
		URL             string `koanf:"url"`
		MaxConns        int    `koanf:"max_conns"`
		MinConns        int    `koanf:"min_conns"`
		MaxConnLifetime string `koanf:"max_conn_lifetime"`
	} `koanf:"database"`

	NATS struct {
		URL string `koanf:"url"`
	} `koanf:"nats"`

	Auth struct {
		JWTSecret string `koanf:"jwt_secret"`
	} `koanf:"auth"`

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

	logger.Info("Starting Tennex Backend",
		zap.String("version", "1.0.0"),
		zap.Int("http_port", config.HTTP.Port),
		zap.Int("grpc_port", config.GRPC.Port))

	// Setup database connection
	dbPool, err := setupDatabase(ctx, struct {
		URL             string
		MaxConns        int
		MinConns        int
		MaxConnLifetime string
	}{
		URL:             config.Database.URL,
		MaxConns:        config.Database.MaxConns,
		MinConns:        config.Database.MinConns,
		MaxConnLifetime: config.Database.MaxConnLifetime,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to setup database", zap.Error(err))
	}
	defer dbPool.Close()

	// Setup NATS connection
	natsConn, err := setupNATS(config.NATS.URL, logger)
	if err != nil {
		logger.Fatal("Failed to setup NATS", zap.Error(err))
	}
	defer natsConn.Close()

	// Create repositories
	eventRepo := repo.NewEventRepository(dbPool)
	outboxRepo := repo.NewOutboxRepository(dbPool)
	accountRepo := repo.NewAccountRepository(dbPool)

	// Create database queries for generated code
	queries := dbgen.New(dbPool)

	// Create core services
	eventService := core.NewEventService(eventRepo, natsConn, logger)
	outboxService := core.NewOutboxService(outboxRepo, eventRepo, logger)
	accountService := core.NewAccountService(accountRepo, logger)

	// Setup servers
	var wg sync.WaitGroup

	// HTTP server
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
		if err := runHTTPServer(ctx, httpConfig, eventService, outboxService, accountService, queries, config.Auth.JWTSecret, logger); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	// gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		grpcConfig := struct {
			Port int
			Host string
		}{
			Port: config.GRPC.Port,
			Host: config.GRPC.Host,
		}
		if err := runGRPCServer(ctx, grpcConfig, eventService, outboxService, accountService, logger); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	// Outbox worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		outboxWorker := core.NewOutboxWorker(outboxService, logger)
		outboxWorker.Start(ctx)
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
	config.HTTP.Port = 8000
	config.HTTP.Host = "0.0.0.0"
	config.GRPC.Port = 6001
	config.GRPC.Host = "0.0.0.0"
	config.Database.URL = "postgres://tennex:tennex123@localhost:5432/tennex?sslmode=disable"
	config.Database.MaxConns = 25
	config.Database.MinConns = 5
	config.Database.MaxConnLifetime = "1h"
	config.NATS.URL = "nats://localhost:4222"
	config.Auth.JWTSecret = "dev-jwt-secret-change-in-production"
	config.Log.Level = "info"
	config.Log.JSON = false

	// Load from file if exists
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		// File doesn't exist, that's okay
	}

	// Load from environment (TENNEX_ prefix)
	if err := k.Load(env.Provider("TENNEX_", ".", func(s string) string {
		// Convert TENNEX_HTTP_PORT to http.port, TENNEX_DATABASE_URL to database.url, etc.
		key := s[7:] // Remove TENNEX_ prefix
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

func setupDatabase(ctx context.Context, dbConfig struct {
	URL             string
	MaxConns        int
	MinConns        int
	MaxConnLifetime string
}, logger *zap.Logger) (*pgxpool.Pool, error) {

	maxConnLifetime, err := time.ParseDuration(dbConfig.MaxConnLifetime)
	if err != nil {
		return nil, fmt.Errorf("invalid max_conn_lifetime: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	poolConfig.MaxConns = int32(dbConfig.MaxConns)
	poolConfig.MinConns = int32(dbConfig.MinConns)
	poolConfig.MaxConnLifetime = maxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.Int("max_conns", dbConfig.MaxConns),
		zap.Int("min_conns", dbConfig.MinConns))

	return pool, nil
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
}, eventService *core.EventService, outboxService *core.OutboxService, accountService *core.AccountService, queries *dbgen.Queries, jwtSecret string, logger *zap.Logger) error {

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

	// API handlers
	apiHandler := handlers.NewAPIHandler(eventService, outboxService, accountService, queries, jwtSecret, logger)
	router.Mount("/", apiHandler.Routes())

	addr := fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	logger.Info("Starting HTTP server", zap.String("addr", addr))

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

	logger.Info("HTTP server stopped")
	return nil
}

func runGRPCServer(ctx context.Context, grpcConfig struct {
	Port int
	Host string
}, eventService *core.EventService, outboxService *core.OutboxService, accountService *core.AccountService, logger *zap.Logger) error {

	addr := fmt.Sprintf("%s:%d", grpcConfig.Host, grpcConfig.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	grpcServer := grpc.NewServer()
	_ = server.NewBridgeServer(eventService, outboxService, accountService, logger)

	// Register service (TODO: replace with generated registration)
	// bridgev1.RegisterBridgeServiceServer(grpcServer, bridgeServer)

	logger.Info("Starting gRPC server", zap.String("addr", addr))

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info("Stopping gRPC server...")
	grpcServer.GracefulStop()

	logger.Info("gRPC server stopped")
	return nil
}
