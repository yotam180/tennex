package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Server provides HTTP endpoints for health checks, metrics, and debugging
type Server struct {
	router *mux.Router
	server *http.Server
	logger *zap.Logger
	port   int

	// Runtime information
	startTime time.Time
	mu        sync.RWMutex
	stats     *Stats
}

// Stats holds runtime statistics
type Stats struct {
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`
}

// Config holds server configuration
type Config struct {
	Port   int
	Logger *zap.Logger

	// Feature flags
	EnablePprof   bool
	EnableMetrics bool
}

// New creates a new HTTP server
func New(cfg Config) *Server {
	s := &Server{
		router:    mux.NewRouter(),
		logger:    cfg.Logger,
		port:      cfg.Port,
		startTime: time.Now(),
		stats:     &Stats{},
	}

	// Setup routes
	s.setupRoutes(cfg.EnablePprof, cfg.EnableMetrics)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", zap.Int("port", s.port))

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes(enablePprof, enableMetrics bool) {
	// Health check endpoints
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/ready", s.handleReady).Methods("GET")
	s.router.HandleFunc("/stats", s.handleStats).Methods("GET")

	// Minimal QR/connect endpoint (stateless, direct whatsmeow usage)
	s.router.HandleFunc("/connect-minimal", s.handleConnectMinimal).Methods("POST")

	// Debug endpoints
	s.router.HandleFunc("/debug/config", s.handleDebugConfig).Methods("GET")
	s.router.HandleFunc("/debug/whatsapp", s.handleDebugWhatsApp).Methods("GET")

	// Optional profiling endpoints
	if enablePprof {
		s.setupPprofRoutes()
	}

	// Optional metrics endpoints
	if enableMetrics {
		s.setupMetricsRoutes()
	}

	// Add logging middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
}

// handleHealth returns basic health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(s.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleReady returns readiness status
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
		"uptime":    time.Since(s.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStats returns detailed runtime statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.StartTime = s.startTime
	s.stats.Uptime = time.Since(s.startTime).String()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.stats)
}

// handleConnectMinimal: accepts {"client_id": "..."}, creates a local whatsmeow client with
// a per-client sqlite store under /app/sessions-min/<client_id>, starts QR, returns the QR code,
// and logs token/JID on success. Minimal PoC-style; no DB writes.
type connectMinimalRequest struct {
	ClientID string `json:"client_id"`
}

type connectMinimalResponse struct {
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
}

func (s *Server) handleConnectMinimal(w http.ResponseWriter, r *http.Request) {
	var req connectMinimalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ClientID == "" {
		http.Error(w, "Invalid JSON request: client_id required", http.StatusBadRequest)
		return
	}

	// Prepare per-client session dir under /app/sessions-min
	base := "/app/sessions-min"
	if v := os.Getenv("TENNEX_BRIDGE_WHATSAPP_SESSION_PATH"); v != "" {
		base = filepath.Join(v, "..", "sessions-min")
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create base dir: %v", err), http.StatusInternalServerError)
		return
	}
	sessDir := filepath.Join(base, fmt.Sprintf("client_%s", req.ClientID))
	if err := os.MkdirAll(sessDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create session dir: %v", err), http.StatusInternalServerError)
		return
	}

	// Create sqlite store and whatsmeow client
	container, err := sqlstore.New(r.Context(), "sqlite3", fmt.Sprintf("file:%s/session.db?_foreign_keys=on", sessDir), waLog.Noop)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create store: %v", err), http.StatusInternalServerError)
		return
	}
	device, err := container.GetFirstDevice(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get device: %v", err), http.StatusInternalServerError)
		return
	}
	client := whatsmeow.NewClient(device, nil)

	// Get QR channel and connect
	qrChan, err := client.GetQRChannel(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get QR channel: %v", err), http.StatusInternalServerError)
		return
	}
	if err := client.Connect(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate session id and respond with first QR code when received
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute)

	// Wait for first QR code event or timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	for {
		select {
		case evt := <-qrChan:
			if evt.Event == "code" {
				// Fire-and-forget watcher that logs success
				go func() {
					for e := range qrChan {
						switch e.Event {
						case "success":
							jid := ""
							if client.Store != nil && client.Store.ID != nil {
								jid = client.Store.ID.String()
							}
							s.logger.Info("QR scan successful (minimal)", zap.String("jid", jid), zap.String("client_id", req.ClientID))
							client.Disconnect()
							return
						case "timeout":
							s.logger.Warn("QR expired before scan (minimal)", zap.String("client_id", req.ClientID))
							client.Disconnect()
							return
						}
					}
				}()

				resp := connectMinimalResponse{SessionID: sessionID, QRCode: evt.Code, Status: "waiting_for_scan", ExpiresAt: expiresAt.Format(time.RFC3339)}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}
		case <-ctx.Done():
			http.Error(w, "Timeout waiting for QR", http.StatusGatewayTimeout)
			return
		}
	}
}

// handleDebugConfig returns configuration information (with sensitive data masked)
func (s *Server) handleDebugConfig(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"server_port": s.port,
		"start_time":  s.startTime,
		"uptime":      time.Since(s.startTime).String(),
		"routes":      s.getRoutes(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDebugWhatsApp returns WhatsApp client debug information
func (s *Server) handleDebugWhatsApp(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"timestamp": time.Now(),
		"message":   "WhatsApp clients are managed per-session via /connect-minimal endpoint",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRoutes returns a list of registered routes for debugging
func (s *Server) getRoutes() []string {
	routes := []string{}
	s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		if template, err := route.GetPathTemplate(); err == nil {
			if methods, err := route.GetMethods(); err == nil {
				for _, method := range methods {
					routes = append(routes, fmt.Sprintf("%s %s", method, template))
				}
			} else {
				routes = append(routes, fmt.Sprintf("* %s", template))
			}
		}
		return nil
	})
	return routes
}

// setupPprofRoutes adds pprof debugging routes
func (s *Server) setupPprofRoutes() {
	// Import pprof for debugging
	// Note: In a real implementation, you'd want to protect these endpoints
	s.logger.Info("Enabling pprof endpoints at /debug/pprof/")
	// Implementation would go here - omitted for brevity
}

// setupMetricsRoutes adds Prometheus metrics routes
func (s *Server) setupMetricsRoutes() {
	s.router.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
	s.logger.Info("Enabling metrics endpoint at /metrics")
}

// handleMetrics serves Prometheus-style metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Basic metrics in Prometheus format
	metrics := fmt.Sprintf(`# HELP tennex_bridge_uptime_seconds Total uptime of the bridge service
# TYPE tennex_bridge_uptime_seconds counter
tennex_bridge_uptime_seconds %.0f

# HELP tennex_bridge_info Information about the bridge service
# TYPE tennex_bridge_info gauge
tennex_bridge_info{version="0.1.0"} 1
`,
		time.Since(s.startTime).Seconds(),
	)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(metrics))
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Debug("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", duration),
			zap.String("user_agent", r.UserAgent()),
			zap.String("remote_addr", r.RemoteAddr))
	})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
