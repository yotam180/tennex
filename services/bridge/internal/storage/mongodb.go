package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// MongoDB handles database operations for client sessions
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	logger   *zap.Logger
}

// ClientSession represents a WhatsApp client session in the database
type ClientSession struct {
	ClientID      string    `bson:"client_id" json:"client_id"`
	SessionID     string    `bson:"session_id" json:"session_id"`
	WhatsAppJID   string    `bson:"whatsapp_jid,omitempty" json:"whatsapp_jid,omitempty"`
	Status        string    `bson:"status" json:"status"` // waiting_for_scan, connected, disconnected, expired
	SessionData   []byte    `bson:"session_data,omitempty" json:"-"` // Encrypted session blob
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	ConnectedAt   *time.Time `bson:"connected_at,omitempty" json:"connected_at,omitempty"`
	DisconnectedAt *time.Time `bson:"disconnected_at,omitempty" json:"disconnected_at,omitempty"`
	ExpiresAt     time.Time `bson:"expires_at" json:"expires_at"`
	LastSeen      time.Time `bson:"last_seen" json:"last_seen"`
}

// ConnectOptions holds MongoDB connection configuration
type ConnectOptions struct {
	URI      string
	Database string
	Logger   *zap.Logger
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(ctx context.Context, opts ConnectOptions) (*MongoDB, error) {
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Set client options
	clientOptions := options.Client().ApplyURI(opts.URI)
	
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(opts.Database)

	mongodb := &MongoDB{
		client:   client,
		database: database,
		logger:   opts.Logger,
	}

	// Create indexes
	if err := mongodb.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	opts.Logger.Info("Connected to MongoDB", 
		zap.String("database", opts.Database))

	return mongodb, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// CreateClientSession creates a new client session
func (m *MongoDB) CreateClientSession(ctx context.Context, session *ClientSession) error {
	collection := m.database.Collection("client_sessions")
	
	session.CreatedAt = time.Now()
	session.LastSeen = time.Now()
	
	_, err := collection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create client session: %w", err)
	}

	m.logger.Debug("Created client session",
		zap.String("client_id", session.ClientID),
		zap.String("session_id", session.SessionID),
		zap.String("status", session.Status))

	return nil
}

// GetClientSession retrieves a client session by session ID
func (m *MongoDB) GetClientSession(ctx context.Context, sessionID string) (*ClientSession, error) {
	collection := m.database.Collection("client_sessions")
	
	var session ClientSession
	err := collection.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get client session: %w", err)
	}

	return &session, nil
}

// GetClientSessionByClientID retrieves the active session for a client
func (m *MongoDB) GetClientSessionByClientID(ctx context.Context, clientID string) (*ClientSession, error) {
	collection := m.database.Collection("client_sessions")
	
	// Find the most recent active session for this client
	opts := options.FindOne().SetSort(bson.D{{"created_at", -1}})
	filter := bson.M{
		"client_id": clientID,
		"status": bson.M{"$in": []string{"waiting_for_scan", "connected"}},
	}
	
	var session ClientSession
	err := collection.FindOne(ctx, filter, opts).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No active session found
		}
		return nil, fmt.Errorf("failed to get client session: %w", err)
	}

	return &session, nil
}

// UpdateClientSession updates a client session
func (m *MongoDB) UpdateClientSession(ctx context.Context, sessionID string, updates bson.M) error {
	collection := m.database.Collection("client_sessions")
	
	updates["last_seen"] = time.Now()
	
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"session_id": sessionID},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update client session: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	m.logger.Debug("Updated client session",
		zap.String("session_id", sessionID),
		zap.Any("updates", updates))

	return nil
}

// MarkSessionConnected marks a session as connected
func (m *MongoDB) MarkSessionConnected(ctx context.Context, sessionID, whatsappJID string) error {
	now := time.Now()
	updates := bson.M{
		"status":       "connected",
		"whatsapp_jid": whatsappJID,
		"connected_at": now,
	}
	
	return m.UpdateClientSession(ctx, sessionID, updates)
}

// MarkSessionDisconnected marks a session as disconnected
func (m *MongoDB) MarkSessionDisconnected(ctx context.Context, sessionID string) error {
	now := time.Now()
	updates := bson.M{
		"status":          "disconnected",
		"disconnected_at": now,
	}
	
	return m.UpdateClientSession(ctx, sessionID, updates)
}

// ExpireOldSessions marks old sessions as expired
func (m *MongoDB) ExpireOldSessions(ctx context.Context) error {
	collection := m.database.Collection("client_sessions")
	
	filter := bson.M{
		"expires_at": bson.M{"$lt": time.Now()},
		"status":     bson.M{"$ne": "expired"},
	}
	
	update := bson.M{
		"$set": bson.M{
			"status":    "expired",
			"last_seen": time.Now(),
		},
	}
	
	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to expire old sessions: %w", err)
	}

	if result.ModifiedCount > 0 {
		m.logger.Info("Expired old sessions", zap.Int64("count", result.ModifiedCount))
	}

	return nil
}

// ListActiveSessions returns all active sessions
func (m *MongoDB) ListActiveSessions(ctx context.Context) ([]*ClientSession, error) {
	collection := m.database.Collection("client_sessions")
	
	filter := bson.M{
		"status": bson.M{"$in": []string{"waiting_for_scan", "connected"}},
	}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*ClientSession
	for cursor.Next(ctx) {
		var session ClientSession
		if err := cursor.Decode(&session); err != nil {
			return nil, fmt.Errorf("failed to decode session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// createIndexes creates necessary database indexes
func (m *MongoDB) createIndexes(ctx context.Context) error {
	collection := m.database.Collection("client_sessions")
	
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"session_id", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"client_id", 1}, {"created_at", -1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"expires_at", 1}},
		},
		{
			Keys: bson.D{{"whatsapp_jid", 1}},
			Options: options.Index().SetSparse(true),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	m.logger.Debug("Created MongoDB indexes")
	return nil
}
