package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	
	"github.com/tennex/bridge/internal/manager"
)

// Server provides HTTP endpoints for health checks, metrics, and debugging
type Server struct {
	router   *mux.Router
	server   *http.Server
	logger   *zap.Logger
	port     int
	
	// Service dependencies
	waClient      WAClient // Legacy single client (for backward compatibility)
	clientManager *manager.ClientManager
	
	// Runtime information
	startTime time.Time
	mu        sync.RWMutex
	stats     *Stats
}

// WAClient interface for WhatsApp client operations
type WAClient interface {
	IsConnected() bool
	GetJID() interface{} // Will handle types.JID conversion internally
}

// Stats holds runtime statistics
type Stats struct {
	StartTime         time.Time `json:"start_time"`
	Uptime            string    `json:"uptime"`
	WhatsAppConnected bool      `json:"whatsapp_connected"`        // Legacy single client
	WhatsAppJID       string    `json:"whatsapp_jid,omitempty"`    // Legacy single client
	ActiveClients     int       `json:"active_clients"`            // Multi-tenant clients
}

// Config holds server configuration
type Config struct {
	Port          int
	Logger        *zap.Logger
	WAClient      WAClient                   // Legacy single client (optional)
	ClientManager *manager.ClientManager    // Multi-tenant client manager
	
	// Feature flags
	EnablePprof  bool
	EnableMetrics bool
}

// New creates a new HTTP server
func New(cfg Config) *Server {
	s := &Server{
		router:        mux.NewRouter(),
		logger:        cfg.Logger,
		port:          cfg.Port,
		waClient:      cfg.WAClient,
		clientManager: cfg.ClientManager,
		startTime:     time.Now(),
		stats:         &Stats{},
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
	
	// Client management endpoints
	if s.clientManager != nil {
		s.router.HandleFunc("/connect-client", s.handleConnectClient).Methods("POST")
	}
	
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
		"status": "healthy",
		"timestamp": time.Now(),
		"uptime": time.Since(s.startTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleReady returns readiness status (includes WhatsApp connection)
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	connected := s.waClient != nil && s.waClient.IsConnected()
	
	response := map[string]interface{}{
		"status": "ready",
		"timestamp": time.Now(),
		"whatsapp_connected": connected,
	}
	
	status := http.StatusOK
	if !connected {
		response["status"] = "not_ready"
		status = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// handleStats returns detailed runtime statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.stats.StartTime = s.startTime
	s.stats.Uptime = time.Since(s.startTime).String()
	
	// Legacy single client stats
	if s.waClient != nil {
		s.stats.WhatsAppConnected = s.waClient.IsConnected()
		if jid := s.waClient.GetJID(); jid != nil {
			s.stats.WhatsAppJID = fmt.Sprintf("%v", jid)
		}
	}
	
	// Multi-tenant client stats
	if s.clientManager != nil {
		s.stats.ActiveClients = s.clientManager.GetActiveClients()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.stats)
}

// handleConnectClient handles POST /connect-client requests
func (s *Server) handleConnectClient(w http.ResponseWriter, r *http.Request) {
	if s.clientManager == nil {
		http.Error(w, "Client manager not available", http.StatusServiceUnavailable)
		return
	}

	var req manager.ConnectClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to decode connect client request", zap.Error(err))
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Validate client ID
	if req.ClientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	// Connect client
	ctx := r.Context()
	response, err := s.clientManager.ConnectClient(ctx, req)
	if err != nil {
		s.logger.Error("Failed to connect client",
			zap.Error(err),
			zap.String("client_id", req.ClientID))
		
		http.Error(w, fmt.Sprintf("Failed to connect client: %v", err), http.StatusInternalServerError)
		return
	}

	s.logger.Info("Client connection initiated",
		zap.String("client_id", req.ClientID),
		zap.String("session_id", response.SessionID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleDebugConfig returns configuration information (with sensitive data masked)
func (s *Server) handleDebugConfig(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"server_port": s.port,
		"start_time": s.startTime,
		"uptime": time.Since(s.startTime).String(),
		"routes": s.getRoutes(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDebugWhatsApp returns WhatsApp client debug information
func (s *Server) handleDebugWhatsApp(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"timestamp": time.Now(),
	}
	
	if s.waClient != nil {
		response["connected"] = s.waClient.IsConnected()
		if jid := s.waClient.GetJID(); jid != nil {
			response["jid"] = fmt.Sprintf("%v", jid)
		}
	} else {
		response["client"] = "not_initialized"
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

# HELP tennex_bridge_whatsapp_connected WhatsApp connection status
# TYPE tennex_bridge_whatsapp_connected gauge
tennex_bridge_whatsapp_connected %d

# HELP tennex_bridge_info Information about the bridge service
# TYPE tennex_bridge_info gauge
tennex_bridge_info{version="0.1.0"} 1
`,
		time.Since(s.startTime).Seconds(),
		func() int {
			if s.waClient != nil && s.waClient.IsConnected() {
				return 1
			}
			return 0
		}(),
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
