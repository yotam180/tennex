package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/core"
	gen "github.com/tennex/pkg/db/gen"
	proto "github.com/tennex/shared/proto/gen/proto"
)

// IntegrationServer implements the integration gRPC service
type IntegrationServer struct {
	proto.UnimplementedIntegrationServiceServer
	integrationService *core.IntegrationService
	db                 *gen.Queries
	logger             *zap.Logger
}

// NewIntegrationServer creates a new integration gRPC server
func NewIntegrationServer(integrationService *core.IntegrationService, db *gen.Queries, logger *zap.Logger) *IntegrationServer {
	return &IntegrationServer{
		integrationService: integrationService,
		db:                 db,
		logger:             logger.Named("integration_server"),
	}
}

// CreateUserIntegration creates a new user integration
func (s *IntegrationServer) CreateUserIntegration(ctx context.Context, req *proto.CreateUserIntegrationRequest) (*proto.CreateUserIntegrationResponse, error) {
	s.logger.Debug("CreateUserIntegration gRPC call received",
		zap.String("user_id", req.UserId),
		zap.String("integration_type", req.IntegrationType))

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		s.logger.Error("Failed to parse user_id as UUID", zap.Error(err))
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	// Convert metadata map to JSON
	var metadataJSON json.RawMessage = []byte("{}")
	if len(req.Metadata) > 0 {
		data, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = data
	}

	// Upsert the integration
	integration, err := s.db.UpsertUserIntegration(ctx, gen.UpsertUserIntegrationParams{
		UserID:          userID,
		IntegrationType: req.IntegrationType,
		ExternalID:      req.PlatformUserId,
		Status:          "connected",
		DisplayName:     req.DisplayName,
		AvatarUrl:       req.AvatarUrl,
		Metadata:        metadataJSON,
		LastSeen:        time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create user integration", zap.Error(err))
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &proto.CreateUserIntegrationResponse{
		Success:           true,
		UserIntegrationId: integration.ID,
	}, nil
}

// UpdateConnectionStatus updates the connection status of an integration
func (s *IntegrationServer) UpdateConnectionStatus(ctx context.Context, req *proto.UpdateConnectionStatusRequest) (*proto.UpdateConnectionStatusResponse, error) {
	s.logger.Debug("UpdateConnectionStatus gRPC call received",
		zap.String("user_id", req.Context.UserId),
		zap.String("integration_type", req.Context.IntegrationType),
		zap.String("status", req.Status.String()))

	userID, err := uuid.Parse(req.Context.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	// Convert proto status to string
	status := convertIntegrationStatusToString(req.Status)

	var lastSeen *time.Time
	if req.Timestamp != nil {
		t := req.Timestamp.AsTime()
		lastSeen = &t
	}

	// Update integration status
	err = s.integrationService.UpdateIntegrationStatus(ctx, userID, req.Context.IntegrationType, status, lastSeen)
	if err != nil {
		s.logger.Error("Failed to update connection status", zap.Error(err))
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	return &proto.UpdateConnectionStatusResponse{
		Success: true,
	}, nil
}

// SyncConversations handles streaming conversation synchronization
func (s *IntegrationServer) SyncConversations(stream proto.IntegrationService_SyncConversationsServer) error {
	s.logger.Debug("SyncConversations stream started")

	var totalProcessed int32
	var batchCount int32

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			return stream.SendAndClose(&proto.SyncConversationsResponse{
				Success:        true,
				ProcessedCount: totalProcessed,
				TotalBatches:   batchCount,
			})
		}
		if err != nil {
			s.logger.Error("Error receiving conversations", zap.Error(err))
			return err
		}

		batchCount++
		s.logger.Debug("Processing conversation batch",
			zap.Int32("batch_number", req.BatchNumber),
			zap.Int("conversations_count", len(req.Conversations)),
			zap.Bool("is_final_batch", req.IsFinalBatch))

		// Process each conversation in the batch
		for _, conv := range req.Conversations {
			err := s.upsertConversation(stream.Context(), req.Context, conv)
			if err != nil {
				s.logger.Error("Failed to upsert conversation",
					zap.String("platform_id", conv.PlatformId),
					zap.Error(err))
				continue
			}
			totalProcessed++
		}

		s.logger.Debug("Processed conversation batch",
			zap.Int32("batch_number", req.BatchNumber),
			zap.Int32("processed_count", totalProcessed))
	}
}

// SyncContacts handles streaming contact synchronization
func (s *IntegrationServer) SyncContacts(stream proto.IntegrationService_SyncContactsServer) error {
	s.logger.Debug("SyncContacts stream started")

	var totalProcessed int32

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&proto.SyncContactsResponse{
				Success:        true,
				ProcessedCount: totalProcessed,
			})
		}
		if err != nil {
			s.logger.Error("Error receiving contacts", zap.Error(err))
			return err
		}

		s.logger.Debug("Processing contact batch",
			zap.Int32("batch_number", req.BatchNumber),
			zap.Int("contacts_count", len(req.Contacts)))

		// Process each contact in the batch
		for _, contact := range req.Contacts {
			err := s.upsertContact(stream.Context(), req.Context, contact)
			if err != nil {
				s.logger.Error("Failed to upsert contact",
					zap.String("platform_id", contact.PlatformId),
					zap.Error(err))
				continue
			}
			totalProcessed++
		}
	}
}

// SyncMessages handles streaming message synchronization
func (s *IntegrationServer) SyncMessages(stream proto.IntegrationService_SyncMessagesServer) error {
	s.logger.Debug("SyncMessages stream started")

	var totalProcessed int32

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&proto.SyncMessagesResponse{
				Success:        true,
				ProcessedCount: totalProcessed,
			})
		}
		if err != nil {
			s.logger.Error("Error receiving messages", zap.Error(err))
			return err
		}

		s.logger.Debug("Processing message batch",
			zap.Int32("batch_number", req.BatchNumber),
			zap.Int("messages_count", len(req.Messages)),
			zap.String("conversation_id", req.ConversationExternalId))

		// Process each message in the batch
		for _, message := range req.Messages {
			err := s.upsertMessage(stream.Context(), req.Context, req.ConversationExternalId, message)
			if err != nil {
				s.logger.Error("Failed to upsert message",
					zap.String("platform_id", message.PlatformId),
					zap.Error(err))
				continue
			}
			totalProcessed++
		}
	}
}

// ProcessMessage handles real-time message processing
func (s *IntegrationServer) ProcessMessage(ctx context.Context, req *proto.ProcessMessageRequest) (*proto.ProcessMessageResponse, error) {
	s.logger.Debug("ProcessMessage gRPC call received",
		zap.String("message_id", req.Message.PlatformId),
		zap.String("conversation_id", req.Message.ConversationId))

	err := s.upsertMessage(ctx, req.Context, req.Message.ConversationId, req.Message)
	if err != nil {
		s.logger.Error("Failed to process message", zap.Error(err))
		return nil, fmt.Errorf("failed to process message: %w", err)
	}

	return &proto.ProcessMessageResponse{
		Success: true,
	}, nil
}

// UpdateConversationState handles conversation state updates
func (s *IntegrationServer) UpdateConversationState(ctx context.Context, req *proto.UpdateConversationStateRequest) (*proto.UpdateConversationStateResponse, error) {
	s.logger.Debug("UpdateConversationState gRPC call received",
		zap.String("conversation_id", req.ConversationExternalId),
		zap.Bool("is_pinned", req.State.IsPinned))

	var muteUntil time.Time
	if req.State.MuteUntil != nil {
		muteUntil = req.State.MuteUntil.AsTime()
	}

	err := s.db.UpdateConversationState(ctx, gen.UpdateConversationStateParams{
		UserIntegrationID:      req.Context.UserIntegrationId,
		ExternalConversationID: req.ConversationExternalId,
		IsArchived:             req.State.IsArchived,
		IsPinned:               req.State.IsPinned,
		IsMuted:                req.State.IsMuted,
		MuteUntil:              muteUntil,
	})
	if err != nil {
		s.logger.Error("Failed to update conversation state", zap.Error(err))
		return nil, fmt.Errorf("failed to update conversation state: %w", err)
	}

	return &proto.UpdateConversationStateResponse{
		Success: true,
	}, nil
}

// Helper functions

func (s *IntegrationServer) upsertConversation(ctx context.Context, integrationCtx *proto.IntegrationContext, conv *proto.Conversation) error {
	// Convert platform metadata
	var platformMetadata json.RawMessage = []byte("{}")
	if len(conv.PlatformMetadata) > 0 {
		data, err := json.Marshal(conv.PlatformMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal platform metadata: %w", err)
		}
		platformMetadata = data
	}

	// Convert timestamps
	var lastMessageAt, lastActivityAt time.Time
	if conv.LastMessageAt != nil {
		lastMessageAt = conv.LastMessageAt.AsTime()
	}
	if conv.LastActivityAt != nil {
		lastActivityAt = conv.LastActivityAt.AsTime()
	}

	var muteUntil time.Time
	if conv.MuteUntil != nil {
		muteUntil = conv.MuteUntil.AsTime()
	}

	// Upsert conversation
	conversation, err := s.db.UpsertConversation(ctx, gen.UpsertConversationParams{
		UserIntegrationID:      integrationCtx.UserIntegrationId,
		ExternalConversationID: conv.PlatformId,
		IntegrationType:        integrationCtx.IntegrationType,
		ConversationType:       convertProtoConversationType(conv.Type),
		Name:                   conv.Name,
		Description:            conv.Description,
		AvatarUrl:              conv.AvatarUrl,
		IsArchived:             conv.IsArchived,
		IsPinned:               conv.IsPinned,
		IsMuted:                conv.IsMuted,
		MuteUntil:              muteUntil,
		IsReadOnly:             conv.IsReadOnly,
		IsLocked:               conv.IsLocked,
		UnreadCount:            conv.UnreadCount,
		UnreadMentionCount:     conv.UnreadMentionCount,
		TotalMessageCount:      conv.TotalMessageCount,
		LastMessageAt:          lastMessageAt,
		LastActivityAt:         lastActivityAt,
		PlatformMetadata:       platformMetadata,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert conversation: %w", err)
	}

	// Upsert participants
	for _, participant := range conv.Participants {
		err := s.upsertConversationParticipant(ctx, conversation.ID, integrationCtx, participant)
		if err != nil {
			s.logger.Error("Failed to upsert participant",
				zap.String("conversation_id", conv.PlatformId),
				zap.String("participant_id", participant.ExternalUserId),
				zap.Error(err))
		}
	}

	return nil
}

func (s *IntegrationServer) upsertConversationParticipant(ctx context.Context, conversationID uuid.UUID, integrationCtx *proto.IntegrationContext, participant *proto.ConversationParticipant) error {
	var platformMetadata json.RawMessage = []byte("{}")
	if len(participant.PlatformMetadata) > 0 {
		data, err := json.Marshal(participant.PlatformMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal participant metadata: %w", err)
		}
		platformMetadata = data
	}

	var joinedAt, leftAt time.Time
	if participant.JoinedAt != nil {
		joinedAt = participant.JoinedAt.AsTime()
	}
	if participant.LeftAt != nil {
		leftAt = participant.LeftAt.AsTime()
	}

	// Default role to 'member' if not specified
	role := participant.Role
	if role == "" {
		role = "member"
	}

	_, err := s.db.UpsertConversationParticipant(ctx, gen.UpsertConversationParticipantParams{
		ConversationID:    conversationID,
		ExternalUserID:    participant.ExternalUserId,
		IntegrationType:   integrationCtx.IntegrationType,
		DisplayName:       participant.DisplayName,
		Role:              role,
		IsActive:          participant.IsActive,
		JoinedAt:          joinedAt,
		LeftAt:            leftAt,
		AddedByExternalID: participant.AddedByExternalId,
		PlatformMetadata:  platformMetadata,
	})
	return err
}

func (s *IntegrationServer) upsertContact(ctx context.Context, integrationCtx *proto.IntegrationContext, contact *proto.Contact) error {
	var platformMetadata json.RawMessage = []byte("{}")
	if len(contact.PlatformMetadata) > 0 {
		data, err := json.Marshal(contact.PlatformMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal contact metadata: %w", err)
		}
		platformMetadata = data
	}

	var lastSeen time.Time
	if contact.LastSeen != nil {
		lastSeen = contact.LastSeen.AsTime()
	}

	_, err := s.db.UpsertContact(ctx, gen.UpsertContactParams{
		UserIntegrationID: integrationCtx.UserIntegrationId,
		ExternalContactID: contact.PlatformId,
		IntegrationType:   integrationCtx.IntegrationType,
		DisplayName:       contact.DisplayName,
		FirstName:         contact.FirstName,
		LastName:          contact.LastName,
		PhoneNumber:       contact.PhoneNumber,
		Username:          contact.Username,
		IsBlocked:         contact.IsBlocked,
		IsFavorite:        contact.IsFavorite,
		LastSeen:          lastSeen,
		AvatarUrl:         contact.AvatarUrl,
		PlatformMetadata:  platformMetadata,
	})
	return err
}

func (s *IntegrationServer) upsertMessage(ctx context.Context, integrationCtx *proto.IntegrationContext, conversationExternalID string, message *proto.Message) error {
	// First, get the conversation ID from external ID
	conversation, err := s.db.GetConversationByExternalID(ctx, gen.GetConversationByExternalIDParams{
		UserIntegrationID:      integrationCtx.UserIntegrationId,
		ExternalConversationID: conversationExternalID,
	})
	if err != nil {
		return fmt.Errorf("failed to find conversation %s: %w", conversationExternalID, err)
	}

	// Convert platform metadata
	var platformMetadata json.RawMessage = []byte("{}")
	if len(message.PlatformMetadata) > 0 {
		data, err := json.Marshal(message.PlatformMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal message metadata: %w", err)
		}
		platformMetadata = data
	}

	// Convert timestamps
	var editTimestamp, deletedAt time.Time
	if message.EditTimestamp != nil {
		editTimestamp = message.EditTimestamp.AsTime()
	}
	if message.DeletedAt != nil {
		deletedAt = message.DeletedAt.AsTime()
	}

	// Upsert message
	msg, err := s.db.UpsertMessage(ctx, gen.UpsertMessageParams{
		ConversationID:    conversation.ID,
		ExternalMessageID: message.PlatformId,
		ExternalServerID:  "", // Not used in this context
		IntegrationType:   integrationCtx.IntegrationType,
		SenderExternalID:  message.SenderId,
		SenderDisplayName: message.SenderDisplayName,
		MessageType:       convertProtoMessageType(message.MessageType),
		Content:           message.Content,
		Timestamp:         message.Timestamp.AsTime(),
		EditTimestamp:     editTimestamp,
		IsFromMe:          message.IsFromMe,
		IsForwarded:       message.IsForwarded,
		IsDeleted:         message.IsDeleted,
		DeletedAt:         deletedAt,
		ReplyToMessageID:  uuid.Nil, // Will implement reply lookup if needed
		ReplyToExternalID: message.ReplyToExternalId,
		DeliveryStatus:    convertProtoMessageStatus(message.Status),
		PlatformMetadata:  platformMetadata,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert message: %w", err)
	}

	// Upsert media attachments
	for _, media := range message.Media {
		err := s.upsertMessageMedia(ctx, msg.ID, media)
		if err != nil {
			s.logger.Error("Failed to upsert message media",
				zap.String("message_id", message.PlatformId),
				zap.Error(err))
		}
	}

	return nil
}

func (s *IntegrationServer) upsertMessageMedia(ctx context.Context, messageID uuid.UUID, media *proto.MessageMedia) error {
	var platformMetadata json.RawMessage = []byte("{}")
	if len(media.PlatformMetadata) > 0 {
		data, err := json.Marshal(media.PlatformMetadata)
		if err != nil {
			return fmt.Errorf("failed to marshal media metadata: %w", err)
		}
		platformMetadata = data
	}

	_, err := s.db.CreateMessageMedia(ctx, gen.CreateMessageMediaParams{
		MessageID:        messageID,
		MediaType:        convertProtoMediaType(media.MediaType),
		FileName:         media.FileName,
		FileSize:         media.FileSize,
		MimeType:         media.MimeType,
		DurationSeconds:  media.DurationSeconds,
		Width:            media.Width,
		Height:           media.Height,
		OriginalUrl:      media.OriginalUrl,
		ThumbnailUrl:     media.ThumbnailUrl,
		LocalFilePath:    "",
		DownloadStatus:   convertProtoDownloadStatus(media.DownloadStatus),
		DownloadedAt:     time.Time{}, // Zero time value
		PlatformMetadata: platformMetadata,
	})
	return err
}

// Conversion helper functions

func convertIntegrationStatusToString(status proto.ConnectionStatus) string {
	switch status {
	case proto.ConnectionStatus_CONNECTION_STATUS_CONNECTED:
		return "connected"
	case proto.ConnectionStatus_CONNECTION_STATUS_CONNECTING:
		return "connecting"
	case proto.ConnectionStatus_CONNECTION_STATUS_DISCONNECTED:
		return "disconnected"
	case proto.ConnectionStatus_CONNECTION_STATUS_ERROR:
		return "error"
	case proto.ConnectionStatus_CONNECTION_STATUS_QR_GENERATED:
		return "connecting"
	case proto.ConnectionStatus_CONNECTION_STATUS_PAIRED:
		return "connecting"
	default:
		return "disconnected"
	}
}

func convertProtoConversationType(convType proto.ConversationType) string {
	switch convType {
	case proto.ConversationType_CONVERSATION_TYPE_INDIVIDUAL:
		return "individual"
	case proto.ConversationType_CONVERSATION_TYPE_GROUP:
		return "group"
	case proto.ConversationType_CONVERSATION_TYPE_BROADCAST:
		return "broadcast"
	case proto.ConversationType_CONVERSATION_TYPE_CHANNEL:
		return "channel"
	default:
		return "individual"
	}
}

func convertProtoMessageType(msgType proto.MessageType) string {
	switch msgType {
	case proto.MessageType_MESSAGE_TYPE_TEXT:
		return "text"
	case proto.MessageType_MESSAGE_TYPE_IMAGE:
		return "image"
	case proto.MessageType_MESSAGE_TYPE_VIDEO:
		return "video"
	case proto.MessageType_MESSAGE_TYPE_AUDIO:
		return "audio"
	case proto.MessageType_MESSAGE_TYPE_DOCUMENT:
		return "document"
	case proto.MessageType_MESSAGE_TYPE_LOCATION:
		return "location"
	case proto.MessageType_MESSAGE_TYPE_CONTACT:
		return "contact"
	case proto.MessageType_MESSAGE_TYPE_STICKER:
		return "sticker"
	case proto.MessageType_MESSAGE_TYPE_POLL:
		return "poll"
	case proto.MessageType_MESSAGE_TYPE_REACTION:
		return "reaction"
	case proto.MessageType_MESSAGE_TYPE_SYSTEM:
		return "system"
	default:
		return "text"
	}
}

func convertProtoMessageStatus(status proto.MessageStatus) string {
	switch status {
	case proto.MessageStatus_MESSAGE_STATUS_SENT:
		return "sent"
	case proto.MessageStatus_MESSAGE_STATUS_DELIVERED:
		return "delivered"
	case proto.MessageStatus_MESSAGE_STATUS_READ:
		return "read"
	case proto.MessageStatus_MESSAGE_STATUS_FAILED:
		return "failed"
	default:
		return "sent"
	}
}

func convertProtoMediaType(mediaType proto.MediaType) string {
	switch mediaType {
	case proto.MediaType_MEDIA_TYPE_IMAGE:
		return "image"
	case proto.MediaType_MEDIA_TYPE_VIDEO:
		return "video"
	case proto.MediaType_MEDIA_TYPE_AUDIO:
		return "audio"
	case proto.MediaType_MEDIA_TYPE_DOCUMENT:
		return "document"
	case proto.MediaType_MEDIA_TYPE_STICKER:
		return "sticker"
	default:
		return "document"
	}
}

func convertProtoDownloadStatus(status proto.DownloadStatus) string {
	switch status {
	case proto.DownloadStatus_DOWNLOAD_STATUS_PENDING:
		return "pending"
	case proto.DownloadStatus_DOWNLOAD_STATUS_DOWNLOADING:
		return "downloading"
	case proto.DownloadStatus_DOWNLOAD_STATUS_COMPLETED:
		return "completed"
	case proto.DownloadStatus_DOWNLOAD_STATUS_FAILED:
		return "failed"
	default:
		return "pending"
	}
}
