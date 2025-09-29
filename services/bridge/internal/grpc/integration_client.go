package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	proto "github.com/tennex/shared/proto/gen/proto"
)

// IntegrationClient wraps the gRPC client for the platform-agnostic integration service
type IntegrationClient struct {
	client proto.IntegrationServiceClient
	conn   *grpc.ClientConn
}

// NewIntegrationClient creates a new integration gRPC client
func NewIntegrationClient(backendAddr string) (*IntegrationClient, error) {
	conn, err := grpc.Dial(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to integration service at %s: %w", backendAddr, err)
	}

	client := proto.NewIntegrationServiceClient(conn)

	return &IntegrationClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (c *IntegrationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// CreateUserIntegration creates a new WhatsApp integration for a user
func (c *IntegrationClient) CreateUserIntegration(ctx context.Context, userID, waJID, displayName, avatarURL string, metadata map[string]string) (int32, error) {
	req := &proto.CreateUserIntegrationRequest{
		UserId:          userID,
		IntegrationType: "whatsapp",
		PlatformUserId:  waJID,
		DisplayName:     displayName,
		AvatarUrl:       avatarURL,
		Metadata:        metadata,
	}

	resp, err := c.client.CreateUserIntegration(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to create user integration: %w", err)
	}

	if !resp.Success {
		return 0, fmt.Errorf("backend reported failure: %s", resp.Error)
	}

	log.Printf("✅ User integration created: user_id=%s, integration_id=%d", userID, resp.UserIntegrationId)
	return resp.UserIntegrationId, nil
}

// UpdateConnectionStatus updates the connection status
func (c *IntegrationClient) UpdateConnectionStatus(ctx context.Context, integrationCtx *proto.IntegrationContext, status proto.ConnectionStatus, qrCode string, metadata map[string]string) error {
	req := &proto.UpdateConnectionStatusRequest{
		Context:   integrationCtx,
		Status:    status,
		QrCode:    qrCode,
		Timestamp: timestamppb.New(time.Now()),
		Metadata:  metadata,
	}

	resp, err := c.client.UpdateConnectionStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("backend reported failure: %s", resp.Error)
	}

	return nil
}

// SyncConversations sends conversations to backend via streaming gRPC
func (c *IntegrationClient) SyncConversations(ctx context.Context, integrationCtx *proto.IntegrationContext, conversations []*proto.Conversation, syncType string) error {
	stream, err := c.client.SyncConversations(ctx)
	if err != nil {
		return fmt.Errorf("failed to create conversations sync stream: %w", err)
	}

	const batchSize = 50
	totalBatches := (len(conversations) + batchSize - 1) / batchSize

	for i := 0; i < len(conversations); i += batchSize {
		end := i + batchSize
		if end > len(conversations) {
			end = len(conversations)
		}

		batch := conversations[i:end]
		isFinal := i+batchSize >= len(conversations)

		req := &proto.SyncConversationsRequest{
			Context:       integrationCtx,
			SyncType:      syncType,
			Conversations: batch,
			IsFinalBatch:  isFinal,
			BatchNumber:   int32(i/batchSize + 1),
		}

		if err := stream.Send(req); err != nil {
			return fmt.Errorf("failed to send conversation batch: %w", err)
		}

		log.Printf("📤 Sent conversation batch %d/%d (%d conversations)", i/batchSize+1, totalBatches, len(batch))
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to close conversations sync stream: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("conversations sync failed: %s", resp.Error)
	}

	log.Printf("✅ Conversations sync completed: %d processed across %d batches", resp.ProcessedCount, resp.TotalBatches)
	return nil
}

// SyncContacts sends contacts to backend via streaming gRPC
func (c *IntegrationClient) SyncContacts(ctx context.Context, integrationCtx *proto.IntegrationContext, contacts []*proto.Contact) error {
	stream, err := c.client.SyncContacts(ctx)
	if err != nil {
		return fmt.Errorf("failed to create contacts sync stream: %w", err)
	}

	const batchSize = 100
	totalBatches := (len(contacts) + batchSize - 1) / batchSize

	for i := 0; i < len(contacts); i += batchSize {
		end := i + batchSize
		if end > len(contacts) {
			end = len(contacts)
		}

		batch := contacts[i:end]
		isFinal := i+batchSize >= len(contacts)

		req := &proto.SyncContactsRequest{
			Context:      integrationCtx,
			Contacts:     batch,
			IsFinalBatch: isFinal,
			BatchNumber:  int32(i/batchSize + 1),
		}

		if err := stream.Send(req); err != nil {
			return fmt.Errorf("failed to send contact batch: %w", err)
		}

		log.Printf("📤 Sent contact batch %d/%d (%d contacts)", i/batchSize+1, totalBatches, len(batch))
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to close contacts sync stream: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("contacts sync failed: %s", resp.Error)
	}

	log.Printf("✅ Contacts sync completed: %d processed", resp.ProcessedCount)
	return nil
}

// SyncMessages sends messages for a conversation to backend via streaming gRPC
func (c *IntegrationClient) SyncMessages(ctx context.Context, integrationCtx *proto.IntegrationContext, conversationID string, messages []*proto.Message) error {
	stream, err := c.client.SyncMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to create messages sync stream: %w", err)
	}

	const batchSize = 200
	totalBatches := (len(messages) + batchSize - 1) / batchSize

	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]
		isFinal := i+batchSize >= len(messages)

		req := &proto.SyncMessagesRequest{
			Context:                integrationCtx,
			ConversationExternalId: conversationID,
			Messages:               batch,
			IsFinalBatch:           isFinal,
			BatchNumber:            int32(i/batchSize + 1),
		}

		if err := stream.Send(req); err != nil {
			return fmt.Errorf("failed to send message batch: %w", err)
		}

		log.Printf("📤 Sent message batch %d/%d (%d messages) for conversation %s", i/batchSize+1, totalBatches, len(batch), conversationID)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to close messages sync stream: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("messages sync failed: %s", resp.Error)
	}

	log.Printf("✅ Messages sync completed for conversation %s: %d processed", conversationID, resp.ProcessedCount)
	return nil
}

// ProcessMessage sends a single real-time message to the backend
func (c *IntegrationClient) ProcessMessage(ctx context.Context, integrationCtx *proto.IntegrationContext, message *proto.Message) error {
	req := &proto.ProcessMessageRequest{
		Context: integrationCtx,
		Message: message,
	}

	resp, err := c.client.ProcessMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to process message: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("message processing failed: %s", resp.Error)
	}

	log.Printf("✅ Real-time message processed: platform_id=%s, internal_id=%s", message.PlatformId, resp.InternalMessageId)
	return nil
}

// UpdateConversationState updates conversation state (pin, mute, archive)
func (c *IntegrationClient) UpdateConversationState(ctx context.Context, integrationCtx *proto.IntegrationContext, conversationID string, state *proto.ConversationState) error {
	req := &proto.UpdateConversationStateRequest{
		Context:                integrationCtx,
		ConversationExternalId: conversationID,
		State:                  state,
	}

	resp, err := c.client.UpdateConversationState(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update conversation state: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("conversation state update failed: %s", resp.Error)
	}

	log.Printf("✅ Conversation state updated: conversation=%s", conversationID)
	return nil
}
