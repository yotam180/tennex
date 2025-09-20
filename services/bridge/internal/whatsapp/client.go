package whatsapp

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
	
	// Import SQLite driver for whatsmeow session storage
	_ "github.com/mattn/go-sqlite3"
)

// Client wraps whatsmeow client with our application logic
type Client struct {
	client    *whatsmeow.Client
	container *sqlstore.Container
	device    *store.Device
	logger    *zap.Logger

	// Event handling
	eventHandler EventHandler
	stopChan     chan struct{}
	wg           sync.WaitGroup

	// Connection state
	connected bool
	mu        sync.RWMutex

	// Configuration
	sessionPath  string
	qrInTerminal bool
}

// EventHandler defines the interface for handling WhatsApp events
type EventHandler interface {
	HandleMessage(msg *events.Message)
	HandleReceipt(receipt *events.Receipt)
	HandlePresence(presence *events.Presence)
	HandleContact(contact *events.Contact)
	HandlePushName(pushName *events.PushName)
	HandleGroupInfo(groupInfo *events.GroupInfo)
	HandleLoggedOut(loggedOut *events.LoggedOut)
	HandleConnected(connected *events.Connected)
	HandleDisconnected(disconnected *events.Disconnected)
}

// ClientConfig holds configuration for the WhatsApp client
type ClientConfig struct {
	SessionPath  string
	QRInTerminal bool
	DBLogLevel   string
	HistorySync  bool
	EventHandler EventHandler
	Logger       *zap.Logger
}

// NewClient creates a new WhatsApp client
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if cfg.EventHandler == nil {
		return nil, fmt.Errorf("event handler is required")
	}

	// Ensure session directory exists
	if err := os.MkdirAll(cfg.SessionPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	// Create SQLite container for session storage
	dbLog := parseDBLogLevel(cfg.DBLogLevel)
	container, err := sqlstore.New(context.Background(), "sqlite3", fmt.Sprintf("file:%s/session.db?_foreign_keys=on", cfg.SessionPath), dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create session container: %w", err)
	}

	// Get the first device from the store
	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	client := &Client{
		container:    container,
		device:       device,
		logger:       cfg.Logger,
		eventHandler: cfg.EventHandler,
		stopChan:     make(chan struct{}),
		sessionPath:  cfg.SessionPath,
		qrInTerminal: cfg.QRInTerminal,
	}

	return client, nil
}

// Connect establishes connection to WhatsApp
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Create whatsmeow client using our device store
	c.client = whatsmeow.NewClient(c.device, nil)

	// Register event handlers
	c.client.AddEventHandler(c.handleEvent)

	c.logger.Info("Connecting to WhatsApp...")

	if c.client.Store.ID == nil {
		// No session exists, need to authenticate
		if err := c.authenticate(ctx); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	} else {
		// Session exists, try to connect
		if err := c.client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	c.connected = true
	c.logger.Info("Successfully connected to WhatsApp",
		zap.String("jid", c.client.Store.ID.String()),
		zap.String("push_name", c.client.Store.PushName))

	return nil
}

// Disconnect closes the WhatsApp connection
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	c.logger.Info("Disconnecting from WhatsApp...")

	close(c.stopChan)
	c.wg.Wait()

	if c.client != nil {
		c.client.Disconnect()
	}

	c.connected = false
	c.logger.Info("Disconnected from WhatsApp")
}

// IsConnected returns the current connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SendMessage sends a text message to the specified JID
func (c *Client) SendMessage(ctx context.Context, to string, message string) (string, error) {
	if !c.IsConnected() {
		return "", fmt.Errorf("not connected to WhatsApp")
	}

	jid, err := types.ParseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	msg := &waProto.Message{
		Conversation: &message,
	}

	resp, err := c.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	c.logger.Debug("Message sent",
		zap.String("to", to),
		zap.String("message_id", resp.ID),
		zap.Time("timestamp", resp.Timestamp))

	return resp.ID, nil
}

// GetJID returns the current user's JID as interface{} for server interface compatibility
func (c *Client) GetJID() interface{} {
	if c.client != nil && c.client.Store.ID != nil {
		return c.client.Store.ID.String()
	}
	return nil
}

// authenticate handles the QR code authentication flow
func (c *Client) authenticate(ctx context.Context) error {
	qrChan, err := c.client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	err = c.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect for QR: %w", err)
	}

	c.logger.Info("Please scan the QR code to authenticate")

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			c.logger.Info("QR Code received", zap.String("code", evt.Code))
			if c.qrInTerminal {
				printQRCode(evt.Code)
			}
		case "timeout":
			c.logger.Error("QR code timeout")
			return fmt.Errorf("QR code authentication timeout")
		case "success":
			c.logger.Info("QR code authentication successful")
			return nil
		default:
			c.logger.Debug("QR code event", zap.String("event", evt.Event))
		}
	}

	return fmt.Errorf("QR authentication failed")
}

// handleEvent is the main event handler that dispatches to specific handlers
func (c *Client) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		c.logger.Debug("Message received",
			zap.String("from", v.Info.Sender.String()),
			zap.String("id", v.Info.ID),
			zap.Time("timestamp", v.Info.Timestamp))
		c.eventHandler.HandleMessage(v)

	case *events.Receipt:
		c.logger.Debug("Receipt received",
			zap.String("type", string(v.Type)),
			zap.String("message_id", v.MessageIDs[0]),
			zap.String("from", v.SourceString()))
		c.eventHandler.HandleReceipt(v)

	case *events.Presence:
		c.eventHandler.HandlePresence(v)

	case *events.Contact:
		c.eventHandler.HandleContact(v)

	case *events.PushName:
		c.eventHandler.HandlePushName(v)

	case *events.GroupInfo:
		c.eventHandler.HandleGroupInfo(v)

	case *events.LoggedOut:
		c.logger.Warn("Logged out from WhatsApp", zap.String("reason", string(v.Reason)))
		c.eventHandler.HandleLoggedOut(v)

	case *events.Connected:
		c.logger.Info("Connected to WhatsApp")
		c.eventHandler.HandleConnected(v)

	case *events.Disconnected:
		c.logger.Warn("Disconnected from WhatsApp")
		c.eventHandler.HandleDisconnected(v)

	default:
		c.logger.Debug("Unhandled event", zap.String("type", fmt.Sprintf("%T", v)))
	}
}

func parseDBLogLevel(level string) waLog.Logger {
	switch level {
	case "TRACE", "DEBUG", "INFO", "WARN", "ERROR":
		return waLog.Noop // Use noop logger for now, can be enhanced later
	default:
		return waLog.Noop // Disable logging
	}
}

// printQRCode prints the QR code to the terminal (placeholder implementation)
func printQRCode(code string) {
	fmt.Printf("\nQR Code: %s\n", code)
	fmt.Println("Please scan this QR code with WhatsApp on your phone")
	fmt.Println("Go to WhatsApp > Settings > Linked Devices > Link a Device")
	fmt.Println("")
}
