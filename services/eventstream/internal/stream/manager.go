package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	// Maximum message queue size per client
	maxQueueSize = 1000

	// Ping interval to keep connections alive
	pingInterval = 30 * time.Second

	// Maximum message size
	maxMessageSize = 32 * 1024 // 32KB
)

// Manager handles WebSocket connections and NATS subscriptions
type Manager struct {
	nats       *nats.Conn
	backendURL string
	logger     *zap.Logger

	// Connection management
	clients map[string]*Client
	mu      sync.RWMutex
}

// Client represents a connected WebSocket client
type Client struct {
	id        string
	accountID string
	conn      *websocket.Conn
	send      chan []byte
	manager   *Manager
	logger    *zap.Logger

	// NATS subscription for this client's account
	subscription *nats.Subscription

	// Context and cancel for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Notification represents a NATS notification message
type Notification struct {
	AccountID string `json:"account_id"`
	NextSeq   int64  `json:"next_seq"`
}

// NewManager creates a new stream manager
func NewManager(natsConn *nats.Conn, backendURL string, logger *zap.Logger) *Manager {
	return &Manager{
		nats:       natsConn,
		backendURL: backendURL,
		logger:     logger.Named("stream_manager"),
		clients:    make(map[string]*Client),
	}
}

// HandleWebSocket handles incoming WebSocket connections
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get account ID from query parameters
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		http.Error(w, "Missing account_id parameter", http.StatusBadRequest)
		return
	}

	// Accept WebSocket connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Allow all origins in development
	})
	if err != nil {
		m.logger.Error("Failed to accept WebSocket connection", zap.Error(err))
		return
	}

	// Create client
	client := m.createClient(accountID, conn)

	m.logger.Info("WebSocket client connected",
		zap.String("client_id", client.id),
		zap.String("account_id", accountID))

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Wait for client to disconnect
	<-client.ctx.Done()

	m.logger.Info("WebSocket client disconnected",
		zap.String("client_id", client.id),
		zap.String("account_id", accountID))
}

// createClient creates a new WebSocket client
func (m *Manager) createClient(accountID string, conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		id:        fmt.Sprintf("%s-%d", accountID, time.Now().UnixNano()),
		accountID: accountID,
		conn:      conn,
		send:      make(chan []byte, maxQueueSize),
		manager:   m,
		logger:    m.logger.With(zap.String("client_id", fmt.Sprintf("%s-*", accountID))),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register client
	m.mu.Lock()
	m.clients[client.id] = client
	m.mu.Unlock()

	// Subscribe to NATS notifications for this account
	subject := fmt.Sprintf("notify.account.%s", accountID)
	sub, err := m.nats.Subscribe(subject, client.handleNotification)
	if err != nil {
		m.logger.Error("Failed to subscribe to NATS",
			zap.String("subject", subject),
			zap.Error(err))
		client.close()
		return client
	}
	client.subscription = sub

	// Start ping ticker
	go client.pingTicker()

	return client
}

// handleNotification handles NATS notifications
func (c *Client) handleNotification(msg *nats.Msg) {
	var notification Notification
	if err := json.Unmarshal(msg.Data, &notification); err != nil {
		c.logger.Error("Failed to unmarshal notification", zap.Error(err))
		return
	}

	// Create WebSocket message
	wsMsg := map[string]interface{}{
		"type":     "notification",
		"next_seq": notification.NextSeq,
	}

	data, err := json.Marshal(wsMsg)
	if err != nil {
		c.logger.Error("Failed to marshal WebSocket message", zap.Error(err))
		return
	}

	// Send to client (non-blocking)
	select {
	case c.send <- data:
		c.logger.Debug("Notification sent to client",
			zap.Int64("next_seq", notification.NextSeq))
	default:
		c.logger.Warn("Client message queue full, dropping notification")
		// Queue is full, client is too slow
		c.close()
	}
}

// writePump sends messages to the WebSocket connection
func (c *Client) writePump() {
	defer c.close()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			if !ok {
				return
			}

			ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
			err := c.conn.Write(ctx, websocket.MessageText, message)
			cancel()

			if err != nil {
				c.logger.Error("Failed to write message", zap.Error(err))
				return
			}
		}
	}
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer c.close()

	// Set read limit
	c.conn.SetReadLimit(maxMessageSize)

	for {
		_, message, err := c.conn.Read(c.ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				c.logger.Debug("Client closed connection normally")
			} else {
				c.logger.Error("Failed to read message", zap.Error(err))
			}
			return
		}

		// Handle client message (for now, just log)
		c.logger.Debug("Received message from client",
			zap.String("message", string(message)))

		// TODO: Handle ping/pong, authentication, etc.
	}
}

// pingTicker sends periodic ping messages
func (c *Client) pingTicker() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
			err := c.conn.Ping(ctx)
			cancel()

			if err != nil {
				c.logger.Error("Failed to ping client", zap.Error(err))
				c.close()
				return
			}
		}
	}
}

// close gracefully closes the client connection
func (c *Client) close() {
	// Cancel context to stop all goroutines
	c.cancel()

	// Unsubscribe from NATS
	if c.subscription != nil {
		c.subscription.Unsubscribe()
	}

	// Close send channel
	close(c.send)

	// Close WebSocket connection
	c.conn.Close(websocket.StatusNormalClosure, "")

	// Remove from manager
	c.manager.mu.Lock()
	delete(c.manager.clients, c.id)
	c.manager.mu.Unlock()
}

// GetClientCount returns the number of connected clients
func (m *Manager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetClientsByAccount returns clients for a specific account
func (m *Manager) GetClientsByAccount(accountID string) []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var clients []*Client
	for _, client := range m.clients {
		if client.accountID == accountID {
			clients = append(clients, client)
		}
	}
	return clients
}
