package grpc

import (
	"context"
	"log"
	"os"

	"github.com/tennex/bridge/internal/recorder"
	proto "github.com/tennex/shared/proto/gen/proto"
)

// RecordingIntegrationClient wraps IntegrationClient with recording capability
type RecordingIntegrationClient struct {
	*IntegrationClient
	recorder *recorder.Recorder
}

// NewRecordingIntegrationClient creates a new recording-enabled client
func NewRecordingIntegrationClient(backendAddr string, recordingsDir string) (*RecordingIntegrationClient, error) {
	client, err := NewIntegrationClient(backendAddr)
	if err != nil {
		return nil, err
	}

	// Check if recording is enabled
	mode := recorder.ModeOff
	if os.Getenv("RECORDING_MODE") == "on" || os.Getenv("RECORDING_MODE") == "record" {
		mode = recorder.ModeRecord
		log.Println("📼 Recording mode enabled")
	}

	rec := recorder.NewRecorder(mode, recordingsDir)

	return &RecordingIntegrationClient{
		IntegrationClient: client,
		recorder:          rec,
	}, nil
}

// StartRecordingSession starts a new recording session
func (c *RecordingIntegrationClient) StartRecordingSession(userID, integrationType string) error {
	return c.recorder.StartSession(userID, integrationType)
}

// EndRecordingSession ends the current recording session
func (c *RecordingIntegrationClient) EndRecordingSession() error {
	return c.recorder.EndSession()
}

// CreateUserIntegration with recording
func (c *RecordingIntegrationClient) CreateUserIntegration(ctx context.Context, userID, waJID, displayName, avatarURL string, metadata map[string]string) (int32, error) {
	req := &proto.CreateUserIntegrationRequest{
		UserId:          userID,
		IntegrationType: "whatsapp",
		PlatformUserId:  waJID,
		DisplayName:     displayName,
		AvatarUrl:       avatarURL,
		Metadata:        metadata,
	}

	// Record the request
	if err := c.recorder.Record(ctx, "CreateUserIntegration", req, map[string]interface{}{
		"user_id":       userID,
		"platform_type": "whatsapp",
	}); err != nil {
		log.Printf("⚠️  Failed to record CreateUserIntegration: %v", err)
	}

	return c.IntegrationClient.CreateUserIntegration(ctx, userID, waJID, displayName, avatarURL, metadata)
}

// UpdateConnectionStatus with recording
func (c *RecordingIntegrationClient) UpdateConnectionStatus(ctx context.Context, integrationCtx *proto.IntegrationContext, status proto.ConnectionStatus, qrCode string, metadata map[string]string) error {
	// Note: We don't record connection status updates as they're not useful for replay
	return c.IntegrationClient.UpdateConnectionStatus(ctx, integrationCtx, status, qrCode, metadata)
}

// SyncConversations with recording
func (c *RecordingIntegrationClient) SyncConversations(ctx context.Context, integrationCtx *proto.IntegrationContext, conversations []*proto.Conversation, syncType string) error {
	// Record the entire batch as a single request (since we want to replay it exactly)
	req := &proto.SyncConversationsRequest{
		Context:       integrationCtx,
		SyncType:      syncType,
		Conversations: conversations,
		IsFinalBatch:  true,
		BatchNumber:   1,
	}

	if err := c.recorder.Record(ctx, "SyncConversations", req, map[string]interface{}{
		"sync_type":          syncType,
		"count":              len(conversations),
		"conversation_count": len(conversations),
	}); err != nil {
		log.Printf("⚠️  Failed to record SyncConversations: %v", err)
	}

	return c.IntegrationClient.SyncConversations(ctx, integrationCtx, conversations, syncType)
}

// SyncContacts with recording
func (c *RecordingIntegrationClient) SyncContacts(ctx context.Context, integrationCtx *proto.IntegrationContext, contacts []*proto.Contact) error {
	req := &proto.SyncContactsRequest{
		Context:      integrationCtx,
		Contacts:     contacts,
		IsFinalBatch: true,
		BatchNumber:  1,
	}

	if err := c.recorder.Record(ctx, "SyncContacts", req, map[string]interface{}{
		"count":         len(contacts),
		"contact_count": len(contacts),
	}); err != nil {
		log.Printf("⚠️  Failed to record SyncContacts: %v", err)
	}

	return c.IntegrationClient.SyncContacts(ctx, integrationCtx, contacts)
}

// SyncMessages with recording
func (c *RecordingIntegrationClient) SyncMessages(ctx context.Context, integrationCtx *proto.IntegrationContext, conversationID string, messages []*proto.Message) error {
	req := &proto.SyncMessagesRequest{
		Context:                integrationCtx,
		ConversationExternalId: conversationID,
		Messages:               messages,
		IsFinalBatch:           true,
		BatchNumber:            1,
	}

	if err := c.recorder.Record(ctx, "SyncMessages", req, map[string]interface{}{
		"conversation_id": conversationID,
		"message_count":   len(messages),
	}); err != nil {
		log.Printf("⚠️  Failed to record SyncMessages: %v", err)
	}

	return c.IntegrationClient.SyncMessages(ctx, integrationCtx, conversationID, messages)
}

// ProcessMessage with recording
func (c *RecordingIntegrationClient) ProcessMessage(ctx context.Context, integrationCtx *proto.IntegrationContext, message *proto.Message) error {
	req := &proto.ProcessMessageRequest{
		Context: integrationCtx,
		Message: message,
	}

	if err := c.recorder.Record(ctx, "ProcessMessage", req, map[string]interface{}{
		"message_id":   message.PlatformId,
		"message_type": message.MessageType,
	}); err != nil {
		log.Printf("⚠️  Failed to record ProcessMessage: %v", err)
	}

	return c.IntegrationClient.ProcessMessage(ctx, integrationCtx, message)
}

// UpdateConversationState with recording
func (c *RecordingIntegrationClient) UpdateConversationState(ctx context.Context, integrationCtx *proto.IntegrationContext, conversationID string, state *proto.ConversationState) error {
	// Note: We don't record state updates for now as they're incremental and less useful for bulk replay
	return c.IntegrationClient.UpdateConversationState(ctx, integrationCtx, conversationID, state)
}

// GetRecorder returns the underlying recorder (for manual session management)
func (c *RecordingIntegrationClient) GetRecorder() *recorder.Recorder {
	return c.recorder
}
