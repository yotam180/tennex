package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waEvents "go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"

	"github.com/tennex/bridge/internal/storage"
	"github.com/tennex/bridge/internal/whatsapp"
)

// ClientManager manages multiple WhatsApp client connections
type ClientManager struct {
	storage      *storage.MongoDB
	logger       *zap.Logger
	eventHandler whatsapp.EventHandler

	// Client management
	clients    map[string]*ManagedClient // sessionID -> client
	clientsMux sync.RWMutex

	// Configuration
	sessionPath string
	dbLogger    waLog.Logger

	// Background tasks
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// ManagedClient wraps a WhatsApp client with session management
type ManagedClient struct {
	SessionID    string
	ClientID     string
	Client       *whatsmeow.Client
	Container    *sqlstore.Container
	Connected    bool
	QRChannel    chan whatsmeow.QRChannelItem
	LastActivity time.Time

	// Synchronization
	mu sync.RWMutex
}

// ConnectClientRequest represents a request to connect a new client
type ConnectClientRequest struct {
	ClientID string `json:"client_id"`
}

// ConnectClientResponse represents the response from connecting a client
type ConnectClientResponse struct {
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
}

// ClientManagerConfig holds configuration for the ClientManager
type ClientManagerConfig struct {
	Storage      *storage.MongoDB
	Logger       *zap.Logger
	EventHandler whatsapp.EventHandler
	SessionPath  string
	DBLogLevel   string
}

// NewClientManager creates a new ClientManager
func NewClientManager(cfg ClientManagerConfig) (*ClientManager, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.EventHandler == nil {
		return nil, fmt.Errorf("event handler is required")
	}

	dbLogger := parseDBLogLevel(cfg.DBLogLevel)

	cm := &ClientManager{
		storage:      cfg.Storage,
		logger:       cfg.Logger,
		eventHandler: cfg.EventHandler,
		clients:      make(map[string]*ManagedClient),
		sessionPath:  cfg.SessionPath,
		dbLogger:     dbLogger,
		stopChan:     make(chan struct{}),
	}

	// Start background tasks
	cm.wg.Add(1)
	go cm.backgroundCleanup()

	return cm, nil
}

// ConnectClient initiates a WhatsApp connection for a client
func (cm *ClientManager) ConnectClient(ctx context.Context, req ConnectClientRequest) (*ConnectClientResponse, error) {
	// Check if client already has an active session
	existingSession, err := cm.storage.GetClientSessionByClientID(ctx, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing sessions: %w", err)
	}

	if existingSession != nil && existingSession.Status == "connected" {
		return nil, fmt.Errorf("client %s already has an active session", req.ClientID)
	}

	// Generate session ID
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(5 * time.Minute) // QR expires in 5 minutes

	// Create database session record
	session := &storage.ClientSession{
		ClientID:  req.ClientID,
		SessionID: sessionID,
		Status:    "waiting_for_scan",
		ExpiresAt: expiresAt,
	}

	if err := cm.storage.CreateClientSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create WhatsApp client
	managedClient, err := cm.createWhatsAppClient(ctx, sessionID, req.ClientID)
	if err != nil {
		// Clean up database session if client creation fails
		cm.storage.UpdateClientSession(ctx, sessionID, map[string]interface{}{
			"status": "error",
		})
		return nil, fmt.Errorf("failed to create WhatsApp client: %w", err)
	}

	// Store managed client
	cm.clientsMux.Lock()
	cm.clients[sessionID] = managedClient
	cm.clientsMux.Unlock()

	// Get QR code
	qrCode, err := cm.getQRCode(ctx, managedClient)
	if err != nil {
		// Clean up on failure
		cm.cleanupClient(sessionID)
		return nil, fmt.Errorf("failed to get QR code: %w", err)
	}

	cm.logger.Info("Created new client connection",
		zap.String("client_id", req.ClientID),
		zap.String("session_id", sessionID))

	return &ConnectClientResponse{
		SessionID: sessionID,
		QRCode:    qrCode,
		Status:    "waiting_for_scan",
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// createWhatsAppClient creates a new WhatsApp client instance
func (cm *ClientManager) createWhatsAppClient(ctx context.Context, sessionID, clientID string) (*ManagedClient, error) {
	// Create session directory for this client
	clientSessionPath := filepath.Join(cm.sessionPath, fmt.Sprintf("client_%s", clientID))

	// Ensure the directory exists
	if err := os.MkdirAll(clientSessionPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	cm.logger.Debug("Created session directory",
		zap.String("path", clientSessionPath),
		zap.String("client_id", clientID))

	// Create SQLite container for this client
	container, err := sqlstore.New(
		ctx,
		"sqlite3",
		fmt.Sprintf("file:%s/session.db?_foreign_keys=on", clientSessionPath),
		cm.dbLogger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session container: %w", err)
	}

	// Get device from the container
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create whatsmeow client
	client := whatsmeow.NewClient(device, nil)

	managedClient := &ManagedClient{
		SessionID:    sessionID,
		ClientID:     clientID,
		Client:       client,
		Container:    container,
		Connected:    false,
		QRChannel:    make(chan whatsmeow.QRChannelItem, 10),
		LastActivity: time.Now(),
	}

	// Set up event handlers
	client.AddEventHandler(func(evt interface{}) {
		cm.handleWhatsAppEvent(managedClient, evt)
	})

	return managedClient, nil
}

// getQRCode initiates QR authentication and returns the QR code
func (cm *ClientManager) getQRCode(ctx context.Context, managedClient *ManagedClient) (string, error) {
	// Get QR channel
	qrChan, err := managedClient.Client.GetQRChannel(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Connect to start QR generation
	err = managedClient.Client.Connect()
	if err != nil {
		// Add detailed logging and persist error on session for diagnostics
		cm.logger.Error("Failed to connect for QR",
			zap.String("session_id", managedClient.SessionID),
			zap.String("client_id", managedClient.ClientID),
			zap.Error(err))

		// Best-effort persist error reason for later inspection
		_ = cm.storage.UpdateClientSession(context.Background(), managedClient.SessionID, map[string]interface{}{
			"status":       "error",
			"error_reason": fmt.Sprintf("connect_error: %v", err),
			"updated_at":   time.Now(),
		})

		return "", fmt.Errorf("failed to connect for QR: %w", err)
	}

	cm.logger.Debug("Waiting for QR code from whatsmeow...",
		zap.String("session_id", managedClient.SessionID),
		zap.String("client_id", managedClient.ClientID))

	// Wait for QR code with timeout - use proper whatsmeow pattern
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Process QR channel events in a loop (like the original whatsapp client)
	for {
		select {
		case evt := <-qrChan:
			cm.logger.Debug("Received QR event",
				zap.String("event", evt.Event),
				zap.String("session_id", managedClient.SessionID))

			switch evt.Event {
			case "code":
				// Return the QR code data immediately
				cm.logger.Info("QR code generated",
					zap.String("session_id", managedClient.SessionID),
					zap.String("code_length", fmt.Sprintf("%d", len(evt.Code))))

				// Mark session state for observability (best-effort)
				_ = cm.storage.UpdateClientSession(context.Background(), managedClient.SessionID, map[string]interface{}{
					"status":     "qr_code_generated",
					"updated_at": time.Now(),
				})

				// Continue watching QR channel events in the background to log outcomes
				go func(sessionID, clientID string, ch <-chan whatsmeow.QRChannelItem) {
					for e := range ch {
						switch e.Event {
						case "success":
							cm.logger.Info("QR code scan successful",
								zap.String("session_id", sessionID),
								zap.String("client_id", clientID))
							_ = cm.storage.UpdateClientSession(context.Background(), sessionID, map[string]interface{}{
								"status":     "scanned",
								"updated_at": time.Now(),
							})
							return
						case "timeout":
							cm.logger.Warn("QR code expired before scan",
								zap.String("session_id", sessionID),
								zap.String("client_id", clientID))
							_ = cm.storage.UpdateClientSession(context.Background(), sessionID, map[string]interface{}{
								"status":     "qr_timeout",
								"updated_at": time.Now(),
							})
							return
						default:
							cm.logger.Debug("QR flow event",
								zap.String("event", e.Event),
								zap.String("session_id", sessionID))
						}
					}
				}(managedClient.SessionID, managedClient.ClientID, qrChan)

				return evt.Code, nil
			case "timeout":
				cm.logger.Warn("QR code timeout", zap.String("session_id", managedClient.SessionID))
				return "", fmt.Errorf("QR code generation timeout")
			case "success":
				// This shouldn't happen in this flow since we return on "code"
				// But if it does, it means immediate connection
				cm.logger.Info("Immediate connection success", zap.String("session_id", managedClient.SessionID))
				return "", fmt.Errorf("client connected immediately, no QR needed")
			default:
				cm.logger.Debug("Unknown QR event",
					zap.String("event", evt.Event),
					zap.String("session_id", managedClient.SessionID))
				// Continue waiting for the "code" event
			}
		case <-ctx.Done():
			cm.logger.Error("Timeout waiting for QR code",
				zap.String("session_id", managedClient.SessionID),
				zap.Error(ctx.Err()))
			return "", fmt.Errorf("timeout waiting for QR code: %w", ctx.Err())
		}
	}
}

// handleWhatsAppEvent processes events from WhatsApp clients
func (cm *ClientManager) handleWhatsAppEvent(managedClient *ManagedClient, evt interface{}) {
	managedClient.mu.Lock()
	managedClient.LastActivity = time.Now()
	managedClient.mu.Unlock()

	switch v := evt.(type) {
	case *waEvents.Connected:
		cm.handleClientConnected(managedClient)
		cm.eventHandler.HandleConnected(v)

	case *waEvents.Disconnected:
		cm.handleClientDisconnected(managedClient)
		cm.eventHandler.HandleDisconnected(v)

	case *waEvents.LoggedOut:
		cm.handleClientLoggedOut(managedClient)
		cm.eventHandler.HandleLoggedOut(v)

	default:
		// Forward other events to the event handler
		// The event handler will need to be enhanced to handle multi-client events
		switch v := evt.(type) {
		case *waEvents.Message:
			cm.eventHandler.HandleMessage(v)
		case *waEvents.Receipt:
			cm.eventHandler.HandleReceipt(v)
		case *waEvents.Presence:
			cm.eventHandler.HandlePresence(v)
		case *waEvents.Contact:
			cm.eventHandler.HandleContact(v)
		case *waEvents.PushName:
			cm.eventHandler.HandlePushName(v)
		case *waEvents.GroupInfo:
			cm.eventHandler.HandleGroupInfo(v)
		}
	}
}

// handleClientConnected processes client connection events
func (cm *ClientManager) handleClientConnected(managedClient *ManagedClient) {
	managedClient.mu.Lock()
	managedClient.Connected = true
	managedClient.mu.Unlock()

	whatsappJID := ""
	if managedClient.Client.Store.ID != nil {
		whatsappJID = managedClient.Client.Store.ID.String()
	}

	// Update database
	ctx := context.Background()
	if err := cm.storage.MarkSessionConnected(ctx, managedClient.SessionID, whatsappJID); err != nil {
		cm.logger.Error("Failed to mark session as connected",
			zap.Error(err),
			zap.String("session_id", managedClient.SessionID))
	}

	cm.logger.Info("Client connected to WhatsApp",
		zap.String("client_id", managedClient.ClientID),
		zap.String("session_id", managedClient.SessionID),
		zap.String("whatsapp_jid", whatsappJID))
}

// handleClientDisconnected processes client disconnection events
func (cm *ClientManager) handleClientDisconnected(managedClient *ManagedClient) {
	managedClient.mu.Lock()
	managedClient.Connected = false
	managedClient.mu.Unlock()

	// Update database
	ctx := context.Background()
	if err := cm.storage.MarkSessionDisconnected(ctx, managedClient.SessionID); err != nil {
		cm.logger.Error("Failed to mark session as disconnected",
			zap.Error(err),
			zap.String("session_id", managedClient.SessionID))
	}

	cm.logger.Info("Client disconnected from WhatsApp",
		zap.String("client_id", managedClient.ClientID),
		zap.String("session_id", managedClient.SessionID))
}

// handleClientLoggedOut processes client logout events
func (cm *ClientManager) handleClientLoggedOut(managedClient *ManagedClient) {
	cm.logger.Warn("Client logged out from WhatsApp",
		zap.String("client_id", managedClient.ClientID),
		zap.String("session_id", managedClient.SessionID))

	// Clean up this client
	cm.cleanupClient(managedClient.SessionID)
}

// cleanupClient removes a client and cleans up resources
func (cm *ClientManager) cleanupClient(sessionID string) {
	cm.clientsMux.Lock()
	defer cm.clientsMux.Unlock()

	managedClient, exists := cm.clients[sessionID]
	if !exists {
		return
	}

	// Disconnect WhatsApp client
	if managedClient.Client != nil {
		managedClient.Client.Disconnect()
	}

	// Close QR channel
	if managedClient.QRChannel != nil {
		close(managedClient.QRChannel)
	}

	// Remove from map
	delete(cm.clients, sessionID)

	cm.logger.Debug("Cleaned up client",
		zap.String("session_id", sessionID),
		zap.String("client_id", managedClient.ClientID))
}

// backgroundCleanup runs background maintenance tasks
func (cm *ClientManager) backgroundCleanup() {
	defer cm.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.performCleanup()
		case <-cm.stopChan:
			return
		}
	}
}

// performCleanup performs maintenance tasks
func (cm *ClientManager) performCleanup() {
	ctx := context.Background()

	// Expire old sessions in database
	if err := cm.storage.ExpireOldSessions(ctx); err != nil {
		cm.logger.Error("Failed to expire old sessions", zap.Error(err))
	}

	// Clean up expired in-memory clients
	cm.clientsMux.Lock()
	var toCleanup []string

	for sessionID, client := range cm.clients {
		client.mu.RLock()
		lastActivity := client.LastActivity
		connected := client.Connected
		client.mu.RUnlock()

		// Clean up clients that haven't been active for 30 minutes and aren't connected
		if !connected && time.Since(lastActivity) > 30*time.Minute {
			toCleanup = append(toCleanup, sessionID)
		}
	}
	cm.clientsMux.Unlock()

	for _, sessionID := range toCleanup {
		cm.cleanupClient(sessionID)
	}

	if len(toCleanup) > 0 {
		cm.logger.Info("Cleaned up inactive clients", zap.Int("count", len(toCleanup)))
	}
}

// Stop stops the ClientManager and cleans up resources
func (cm *ClientManager) Stop() {
	close(cm.stopChan)
	cm.wg.Wait()

	// Clean up all clients
	cm.clientsMux.Lock()
	for sessionID := range cm.clients {
		cm.cleanupClient(sessionID)
	}
	cm.clientsMux.Unlock()

	cm.logger.Info("ClientManager stopped")
}

// GetActiveClients returns the number of active clients
func (cm *ClientManager) GetActiveClients() int {
	cm.clientsMux.RLock()
	defer cm.clientsMux.RUnlock()
	return len(cm.clients)
}

// parseDBLogLevel converts log level string to waLog.Logger
func parseDBLogLevel(level string) waLog.Logger {
	// Use noop logger for now, can be enhanced later
	return waLog.Noop
}
