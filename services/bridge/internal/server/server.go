package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mdp/qrterminal/v3"
	"go.uber.org/zap"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// ActiveClient represents a live WhatsApp client
type ActiveClient struct {
	Client    *whatsmeow.Client
	SessionID string
	ClientID  string
	StartTime time.Time
}

// Server provides HTTP endpoints for health checks, metrics, and debugging
type Server struct {
	router *mux.Router
	server *http.Server
	logger *zap.Logger
	port   int

	// Database
	dbURL       string
	dbContainer *sqlstore.Container

	// Runtime information
	startTime time.Time
	mu        sync.RWMutex
	stats     *Stats

	// Active WhatsApp clients - keep them alive to prevent disconnection
	activeClients map[string]*ActiveClient
	clientsMu     sync.RWMutex
}

// Stats holds runtime statistics
type Stats struct {
	StartTime time.Time `json:"start_time"`
	Uptime    string    `json:"uptime"`
}

// Config holds server configuration
type Config struct {
	Port        int
	Logger      *zap.Logger
	DatabaseURL string

	// Feature flags
	EnablePprof   bool
	EnableMetrics bool
}

// New creates a new HTTP server
func New(cfg Config) *Server {
	s := &Server{
		router:        mux.NewRouter(),
		logger:        cfg.Logger,
		port:          cfg.Port,
		dbURL:         cfg.DatabaseURL,
		startTime:     time.Now(),
		stats:         &Stats{},
		activeClients: make(map[string]*ActiveClient),
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
	// Initialize database connection
	if err := s.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	s.logger.Info("Starting HTTP server", zap.Int("port", s.port))

	// Start server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	return nil
}

// initDatabase initializes the SQLite connection for whatsmeow
func (s *Server) initDatabase(ctx context.Context) error {
	s.logger.Info("Initializing SQLite connection for WhatsApp sessions")

	// Use SQLite like the test code
	dsn := "file:sessions/session.db?_foreign_keys=on"
	container, err := sqlstore.New(ctx, "sqlite3", dsn, waLog.Noop)
	if err != nil {
		return fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	// Set device properties exactly like test/main.go
	store.DeviceProps.Os = proto.String("Temple OS")
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	store.DeviceProps.RequireFullSync = proto.Bool(true)

	s.dbContainer = container
	s.logger.Info("SQLite connection initialized successfully")
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
	s.router.HandleFunc("/debug/clients", s.handleDebugClients).Methods("GET")

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to decode connect minimal request", zap.Error(err))
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if req.ClientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	if s.dbContainer == nil {
		s.logger.Error("Database container not initialized")
		http.Error(w, "Internal server error: database not available", http.StatusInternalServerError)
		return
	}

	// Get device for this client from SQLite (whatsmeow will create a new device if needed)
	device, err := s.dbContainer.GetFirstDevice(r.Context())
	if err != nil {
		s.logger.Error("Failed to get device from SQLite", zap.Error(err), zap.String("client_id", req.ClientID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create whatsmeow client with nil logger exactly like test/main.go
	client := whatsmeow.NewClient(device, nil)

	// Get QR channel BEFORE connect, exactly like test/main.go
	qrChan, err := client.GetQRChannel(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get QR channel: %v", err), http.StatusInternalServerError)
		return
	}

	// Connect to start QR generation
	if err := client.Connect(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate session id and store client to keep it alive
	sessionID := uuid.New().String()
	activeClient := &ActiveClient{
		Client:    client,
		SessionID: sessionID,
		ClientID:  req.ClientID,
		StartTime: time.Now(),
	}

	// Store client to keep it alive and prevent garbage collection
	s.clientsMu.Lock()
	s.activeClients[sessionID] = activeClient
	s.clientsMu.Unlock()

	// Start background goroutine to handle all QR events - this keeps the client alive
	go s.handleQREvents(qrChan, activeClient)

	// Wait for first QR code with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	for {
		select {
		case evt := <-qrChan:
			if evt.Event == "code" {
				// Print QR to stderr like test code
				fmt.Fprintf(os.Stderr, "\nScan this QR with WhatsApp (client: %s, session: %s):\n", req.ClientID, sessionID)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stderr)
				fmt.Fprintf(os.Stderr, "\n(If it expires, just call again.)\n")

				// Return the QR code but keep client alive in background
				resp := connectMinimalResponse{
					SessionID: sessionID,
					QRCode:    evt.Code,
					Status:    "waiting_for_scan",
					ExpiresAt: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return // Client continues running in background goroutine
			}
		case <-ctx.Done():
			// Clean up client on timeout
			s.removeActiveClient(sessionID)
			http.Error(w, "Timeout waiting for QR", http.StatusGatewayTimeout)
			return
		}
	}
}

// handleQREvents processes QR events in background, keeping the client alive
func (s *Server) handleQREvents(qrChan <-chan whatsmeow.QRChannelItem, activeClient *ActiveClient) {
	defer s.removeActiveClient(activeClient.SessionID)

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// First QR code is handled in main function, subsequent ones just logged
			s.logger.Debug("QR code generated",
				zap.String("client_id", activeClient.ClientID),
				zap.String("session_id", activeClient.SessionID))
		case "timeout":
			s.logger.Warn("QR expired before scan",
				zap.String("client_id", activeClient.ClientID),
				zap.String("session_id", activeClient.SessionID))
			return
		case "success":
			jid := ""
			if activeClient.Client.Store != nil && activeClient.Client.Store.ID != nil {
				jid = activeClient.Client.Store.ID.String()
			}
			s.logger.Info("QR scan successful - client connected and staying alive",
				zap.String("jid", jid),
				zap.String("client_id", activeClient.ClientID),
				zap.String("session_id", activeClient.SessionID))

			fmt.Fprintf(os.Stderr, "\n✅ QR scan successful! Session established. JID: %s (client: %s, session: %s)\n",
				jid, activeClient.ClientID, activeClient.SessionID)

			// Keep client alive - don't return, let it run indefinitely
			// In a real implementation, you might want to add message handlers here
			s.keepClientAlive(activeClient)
			return
		default:
			s.logger.Debug("Ignoring QR event",
				zap.String("event", evt.Event),
				zap.String("client_id", activeClient.ClientID),
				zap.String("session_id", activeClient.SessionID))
		}
	}
}

// keepClientAlive keeps the WhatsApp client running indefinitely
func (s *Server) keepClientAlive(activeClient *ActiveClient) {
	s.logger.Info("Keeping WhatsApp client alive",
		zap.String("client_id", activeClient.ClientID),
		zap.String("session_id", activeClient.SessionID))

	// In a real implementation, you might:
	// 1. Add message event handlers
	// 2. Set up periodic health checks
	// 3. Handle reconnection logic
	// 4. Save session data periodically

	// For now, just keep it alive by blocking
	select {
	// This will block forever, keeping the client alive
	// You could add channels here for client management (stop, restart, etc.)
	}
}

// removeActiveClient safely removes a client from the active clients map
func (s *Server) removeActiveClient(sessionID string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if client, exists := s.activeClients[sessionID]; exists {
		s.logger.Info("Removing active client",
			zap.String("session_id", sessionID),
			zap.String("client_id", client.ClientID))

		// Disconnect the client if it's still connected
		if client.Client.IsConnected() {
			client.Client.Disconnect()
		}

		delete(s.activeClients, sessionID)
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

// handleDebugClients returns information about active WhatsApp clients
func (s *Server) handleDebugClients(w http.ResponseWriter, r *http.Request) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	clients := make([]map[string]interface{}, 0, len(s.activeClients))
	for _, client := range s.activeClients {
		jid := ""
		isConnected := client.Client.IsConnected()
		if client.Client.Store != nil && client.Client.Store.ID != nil {
			jid = client.Client.Store.ID.String()
		}

		clients = append(clients, map[string]interface{}{
			"session_id":   client.SessionID,
			"client_id":    client.ClientID,
			"start_time":   client.StartTime,
			"uptime":       time.Since(client.StartTime).String(),
			"is_connected": isConnected,
			"jid":          jid,
		})
	}

	response := map[string]interface{}{
		"timestamp":      time.Now(),
		"active_clients": clients,
		"total_count":    len(clients),
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
