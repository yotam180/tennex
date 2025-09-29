package whatsapp

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/types/known/timestamppb"

	backendGRPC "github.com/tennex/bridge/internal/grpc"
	proto "github.com/tennex/shared/proto/gen/proto"
)

// EventsProcessor handles WhatsApp events and sends them to the backend
type EventsProcessor struct {
	integrationClient *backendGRPC.RecordingIntegrationClient
	backendClient     *backendGRPC.BackendClient // Keep old client for compatibility
	userID            string
	userIntegrationID int32
	integrationCtx    *proto.IntegrationContext
}

// NewEventsProcessor creates a new events processor
func NewEventsProcessor(integrationClient *backendGRPC.RecordingIntegrationClient, backendClient *backendGRPC.BackendClient, userID string) *EventsProcessor {
	return &EventsProcessor{
		integrationClient: integrationClient,
		backendClient:     backendClient,
		userID:            userID,
	}
}

// SetIntegrationContext sets the integration context after user integration is created
func (p *EventsProcessor) SetIntegrationContext(userIntegrationID int32, waJID string) {
	p.userIntegrationID = userIntegrationID
	p.integrationCtx = &proto.IntegrationContext{
		UserId:            p.userID,
		UserIntegrationId: userIntegrationID,
		IntegrationType:   "whatsapp",
		PlatformUserId:    waJID,
	}
}

// ProcessEvent processes a WhatsApp event and sends it to the backend
// On any error, it will panic to force disconnection for easier debugging
func (p *EventsProcessor) ProcessEvent(ctx context.Context, evt interface{}) {
	// Get event type name
	eventType := reflect.TypeOf(evt).String()
	log.Printf("\n🔔 Processing WhatsApp event: %s", eventType)

	var err error

	// Handle specific event types
	switch v := evt.(type) {
	case *events.Connected:
		err = p.handleConnected(ctx, v)

	case *events.Disconnected:
		err = p.handleDisconnected(ctx, v)

	case *events.LoggedOut:
		err = p.handleLoggedOut(ctx, v)

	case *events.HistorySync:
		err = p.handleHistorySync(ctx, v)

	case *events.Message:
		err = p.handleMessage(ctx, v)

	case *events.Receipt:
		err = p.handleReceipt(ctx, v)

	case *events.AppStateSyncComplete:
		err = p.handleAppStateSyncComplete(ctx, v)

	case *events.Contact:
		err = p.handleContact(ctx, v)

	case *events.PushName:
		err = p.handlePushName(ctx, v)

	case *events.GroupInfo:
		err = p.handleGroupInfo(ctx, v)

	case *events.JoinedGroup:
		err = p.handleJoinedGroup(ctx, v)

	case *events.Presence:
		err = p.handlePresence(ctx, v)

	case *events.ChatPresence:
		err = p.handleChatPresence(ctx, v)

	// Note: OfflineSyncPreview and OfflineSyncCompleted events don't exist in this whatsmeow version

	default:
		log.Printf("❓ Unhandled WhatsApp event type: %s", eventType)
		return
	}

	// FAIL FAST: Panic on first error to force disconnection for debugging
	if err != nil {
		log.Printf("🚨 FATAL ERROR processing %s: %v", eventType, err)
		log.Printf("🚨 Panicking to force disconnection for easier debugging")
		panic(fmt.Sprintf("Event processing failed: %v", err))
	}
}

func (p *EventsProcessor) handleConnected(ctx context.Context, evt *events.Connected) error {
	log.Printf("🔗 WhatsApp Connected!")

	if p.integrationCtx != nil {
		err := p.integrationClient.UpdateConnectionStatus(
			ctx,
			p.integrationCtx,
			proto.ConnectionStatus_CONNECTION_STATUS_CONNECTED,
			"",
			map[string]string{
				"event_type": "connected",
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update connection status to connected: %w", err)
		}
	}
	return nil
}

func (p *EventsProcessor) handleDisconnected(ctx context.Context, evt *events.Disconnected) error {
	log.Printf("❌ WhatsApp Disconnected")

	if p.integrationCtx != nil {
		err := p.integrationClient.UpdateConnectionStatus(
			ctx,
			p.integrationCtx,
			proto.ConnectionStatus_CONNECTION_STATUS_DISCONNECTED,
			"",
			map[string]string{
				"reason": "disconnected",
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update connection status to disconnected: %w", err)
		}
	}
	return nil
}

func (p *EventsProcessor) handleLoggedOut(ctx context.Context, evt *events.LoggedOut) error {
	log.Printf("👋 WhatsApp Logged Out: reason=%s", evt.Reason)

	if p.integrationCtx != nil {
		err := p.integrationClient.UpdateConnectionStatus(
			ctx,
			p.integrationCtx,
			proto.ConnectionStatus_CONNECTION_STATUS_DISCONNECTED,
			"",
			map[string]string{
				"reason":        "logged_out",
				"logout_reason": evt.Reason.String(),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update connection status to logged out: %w", err)
		}
	}
	return nil
}

func (p *EventsProcessor) handleHistorySync(ctx context.Context, evt *events.HistorySync) error {
	log.Printf("🔄 History Sync: type=%s, conversations=%d",
		evt.Data.SyncType, len(evt.Data.Conversations))

	if p.integrationCtx == nil {
		log.Printf("⚠️  Integration context not set, skipping history sync")
		return nil
	}

	// Process conversations
	if len(evt.Data.Conversations) > 0 {
		conversations := make([]*proto.Conversation, 0, len(evt.Data.Conversations))

		for _, waConv := range evt.Data.Conversations {
			protoConv := p.convertHistorySyncConversation(waConv)
			if protoConv != nil {
				conversations = append(conversations, protoConv)
			}
		}

		if len(conversations) > 0 {
			err := p.integrationClient.SyncConversations(ctx, p.integrationCtx, conversations, evt.Data.SyncType.String())
			if err != nil {
				return fmt.Errorf("failed to sync %d conversations: %w", len(conversations), err)
			}
			log.Printf("✅ Synced %d conversations from history", len(conversations))
		}
	}

	// Process messages from conversations
	var totalMessages int
	messagesByConversation := make(map[string][]*proto.Message)

	for _, waConv := range evt.Data.Conversations {
		for _, waMsg := range waConv.Messages {
			protoMsg := p.convertHistorySyncMessage(waMsg)
			if protoMsg != nil {
				conversationID := protoMsg.ConversationId
				messagesByConversation[conversationID] = append(messagesByConversation[conversationID], protoMsg)
				totalMessages++
			}
		}
	}

	// Sync messages for each conversation
	if totalMessages > 0 {
		log.Printf("🔄 Processing %d messages from %d conversations", totalMessages, len(messagesByConversation))
		for conversationID, messages := range messagesByConversation {
			err := p.integrationClient.SyncMessages(ctx, p.integrationCtx, conversationID, messages)
			if err != nil {
				return fmt.Errorf("failed to sync %d messages for conversation %s: %w", len(messages), conversationID, err)
			}
			log.Printf("✅ Synced %d messages for conversation %s from history", len(messages), conversationID)
		}
	}
	return nil
}

func (p *EventsProcessor) handleMessage(ctx context.Context, evt *events.Message) error {
	log.Printf("📨 New Message: ID=%s, from=%s, chat=%s",
		evt.Info.ID, evt.Info.Sender.String(), evt.Info.Chat.String())

	if p.integrationCtx == nil {
		log.Printf("⚠️  Integration context not set, skipping message")
		return nil
	}

	protoMsg := p.convertMessage(evt)
	if protoMsg == nil {
		log.Printf("⚠️  Failed to convert message")
		return nil
	}

	err := p.integrationClient.ProcessMessage(ctx, p.integrationCtx, protoMsg)
	if err != nil {
		return fmt.Errorf("failed to process real-time message: %w", err)
	}
	return nil
}

func (p *EventsProcessor) handleReceipt(ctx context.Context, evt *events.Receipt) error {
	log.Printf("✅ Message Receipt: type=%s, messages=%v, sender=%s",
		evt.Type, evt.MessageIDs, evt.SourceString())
	return nil
}

func (p *EventsProcessor) handleAppStateSyncComplete(ctx context.Context, evt *events.AppStateSyncComplete) error {
	log.Printf("🔄 App State Sync Complete: %s", evt.Name)
	return nil
}

func (p *EventsProcessor) handleContact(ctx context.Context, evt *events.Contact) error {
	log.Printf("👤 Contact Update: JID=%s", evt.JID.String())

	if p.integrationCtx == nil {
		log.Printf("⚠️  Integration context not set, skipping contact")
		return nil
	}

	protoContact := p.convertContact(evt)
	if protoContact == nil {
		return nil
	}

	// Send single contact as a batch
	contacts := []*proto.Contact{protoContact}
	err := p.integrationClient.SyncContacts(ctx, p.integrationCtx, contacts)
	if err != nil {
		return fmt.Errorf("failed to sync contact update: %w", err)
	}
	return nil
}

func (p *EventsProcessor) handlePushName(ctx context.Context, evt *events.PushName) error {
	log.Printf("📛 Push Name Update: JID=%s, name=%s", evt.JID.String(), evt.Message.PushName)
	return nil
}

func (p *EventsProcessor) handleGroupInfo(ctx context.Context, evt *events.GroupInfo) error {
	log.Printf("👥 Group Info Update: JID=%s, name=%s", evt.JID.String(), evt.Name.Name)
	return nil
}

func (p *EventsProcessor) handleJoinedGroup(ctx context.Context, evt *events.JoinedGroup) error {
	log.Printf("🎉 Joined Group: JID=%s, reason=%s", evt.JID.String(), evt.Reason)
	return nil
}

func (p *EventsProcessor) handlePresence(ctx context.Context, evt *events.Presence) error {
	log.Printf("👁️  Presence Update: from=%s, last_seen=%v", evt.From.String(), evt.LastSeen)
	return nil
}

func (p *EventsProcessor) handleChatPresence(ctx context.Context, evt *events.ChatPresence) error {
	log.Printf("👥 Chat Presence Update: chat=%s, participants=%v", evt.Chat.String(), evt.State)
	return nil
}

// Conversion helpers
func (p *EventsProcessor) convertHistorySyncConversation(waConv *waHistorySync.Conversation) *proto.Conversation {
	if waConv == nil {
		return nil
	}

	conv := &proto.Conversation{
		PlatformId:       getStringPtr(waConv.ID),
		Name:             getStringPtr(waConv.Name),
		IsArchived:       getBoolPtr(waConv.Archived),
		IsPinned:         getUint32Ptr(waConv.Pinned) > 0,
		IsReadOnly:       getBoolPtr(waConv.ReadOnly),
		UnreadCount:      int32(getUint32Ptr(waConv.UnreadCount)),
		PlatformMetadata: make(map[string]string),
	}

	// Handle mute status
	if waConv.MuteEndTime != nil && *waConv.MuteEndTime > uint64(time.Now().Unix()) {
		conv.IsMuted = true
		conv.MuteUntil = timestamppb.New(time.Unix(int64(*waConv.MuteEndTime), 0))
	}

	// Set conversation type based on participant count
	if len(waConv.Participant) > 0 {
		conv.Type = proto.ConversationType_CONVERSATION_TYPE_GROUP

		// Add participants
		for _, participant := range waConv.Participant {
			conv.Participants = append(conv.Participants, &proto.ConversationParticipant{
				ExternalUserId: getStringPtr(participant.UserJID),
				DisplayName:    "",   // DisplayName not available in GroupParticipant
				IsActive:       true, // IsDeleted not available, assume active
			})
		}
	} else {
		conv.Type = proto.ConversationType_CONVERSATION_TYPE_INDIVIDUAL
	}

	// Convert timestamps
	if waConv.ConversationTimestamp != nil {
		conv.LastActivityAt = timestamppb.New(time.Unix(int64(*waConv.ConversationTimestamp), 0))
	}
	if waConv.LastMsgTimestamp != nil {
		conv.LastMessageAt = timestamppb.New(time.Unix(int64(*waConv.LastMsgTimestamp), 0))
	}

	// Add platform-specific metadata
	conv.PlatformMetadata["pinned_timestamp"] = strconv.FormatUint(uint64(getUint32Ptr(waConv.Pinned)), 10)
	conv.PlatformMetadata["end_of_history_transfer"] = fmt.Sprintf("%v", getBoolPtr(waConv.EndOfHistoryTransfer))
	if waConv.DisappearingMode != nil {
		conv.PlatformMetadata["disappearing_mode"] = fmt.Sprintf("%v", waConv.DisappearingMode)
	}

	return conv
}

func (p *EventsProcessor) convertHistorySyncMessage(waMsg *waHistorySync.HistorySyncMsg) *proto.Message {
	if waMsg == nil || waMsg.Message == nil {
		return nil
	}

	webMsg := waMsg.Message
	msg := &proto.Message{
		PlatformId:       getStringPtr(webMsg.Key.ID),
		ConversationId:   getStringPtr(webMsg.Key.RemoteJID),
		SenderId:         getStringPtr(webMsg.Key.Participant),
		Timestamp:        timestamppb.New(time.Unix(int64(getUint64Ptr(webMsg.MessageTimestamp)), 0)),
		IsFromMe:         getBoolPtr(webMsg.Key.FromMe),
		PlatformMetadata: make(map[string]string),
	}

	// If no participant, use the sender from the remote JID
	if msg.SenderId == "" {
		msg.SenderId = msg.ConversationId
	}

	// Determine message type and content from the message content
	if webMsg.Message != nil {
		if webMsg.Message.Conversation != nil {
			msg.MessageType = proto.MessageType_MESSAGE_TYPE_TEXT
			msg.Content = getStringPtr(webMsg.Message.Conversation)
		} else if webMsg.Message.ImageMessage != nil {
			msg.MessageType = proto.MessageType_MESSAGE_TYPE_IMAGE
			msg.Content = getStringPtr(webMsg.Message.ImageMessage.Caption)
		} else if webMsg.Message.VideoMessage != nil {
			msg.MessageType = proto.MessageType_MESSAGE_TYPE_VIDEO
			msg.Content = getStringPtr(webMsg.Message.VideoMessage.Caption)
		} else if webMsg.Message.AudioMessage != nil {
			msg.MessageType = proto.MessageType_MESSAGE_TYPE_AUDIO
		} else if webMsg.Message.DocumentMessage != nil {
			msg.MessageType = proto.MessageType_MESSAGE_TYPE_DOCUMENT
			msg.Content = getStringPtr(webMsg.Message.DocumentMessage.Title)
		} else {
			msg.MessageType = proto.MessageType_MESSAGE_TYPE_TEXT
			msg.Content = "[Unsupported message type]"
		}
	} else {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_TEXT
		msg.Content = "[Empty message]"
	}

	// Add platform metadata
	if webMsg.Status != nil {
		msg.PlatformMetadata["status"] = webMsg.Status.String()
	}

	return msg
}

func (p *EventsProcessor) convertMessage(evt *events.Message) *proto.Message {
	if evt == nil {
		return nil
	}

	msg := &proto.Message{
		PlatformId:       evt.Info.ID,
		ConversationId:   evt.Info.Chat.String(),
		SenderId:         evt.Info.Sender.String(),
		Timestamp:        timestamppb.New(evt.Info.Timestamp),
		IsFromMe:         evt.Info.IsFromMe,
		PlatformMetadata: make(map[string]string),
	}

	// Determine message type and content
	if evt.Message.GetConversation() != "" {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_TEXT
		msg.Content = evt.Message.GetConversation()
	} else if evt.Message.GetImageMessage() != nil {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_IMAGE
		if evt.Message.GetImageMessage().GetCaption() != "" {
			msg.Content = evt.Message.GetImageMessage().GetCaption()
		}
	} else if evt.Message.GetVideoMessage() != nil {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_VIDEO
		if evt.Message.GetVideoMessage().GetCaption() != "" {
			msg.Content = evt.Message.GetVideoMessage().GetCaption()
		}
	} else if evt.Message.GetAudioMessage() != nil {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_AUDIO
	} else if evt.Message.GetDocumentMessage() != nil {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_DOCUMENT
		if evt.Message.GetDocumentMessage().GetTitle() != "" {
			msg.Content = evt.Message.GetDocumentMessage().GetTitle()
		}
	} else {
		msg.MessageType = proto.MessageType_MESSAGE_TYPE_TEXT
		msg.Content = "[Unsupported message type]"
	}

	// Add platform metadata
	msg.PlatformMetadata["server_id"] = strconv.Itoa(int(evt.Info.ServerID))
	msg.PlatformMetadata["push_name"] = evt.Info.PushName

	return msg
}

func (p *EventsProcessor) convertContact(evt *events.Contact) *proto.Contact {
	if evt == nil || evt.Action == nil {
		return nil
	}

	contact := &proto.Contact{
		PlatformId:       evt.JID.String(),
		DisplayName:      evt.Action.GetFullName(),
		FirstName:        evt.Action.GetFirstName(),
		PlatformMetadata: make(map[string]string),
	}

	// Add phone number if available
	if evt.JID.Server == types.DefaultUserServer {
		// Extract phone number from JID (everything before @)
		if idx := strings.Index(evt.JID.String(), "@"); idx > 0 {
			contact.PhoneNumber = evt.JID.String()[:idx]
		}
	}

	// Add platform metadata
	contact.PlatformMetadata["lid_jid"] = evt.Action.GetLidJID()

	return contact
}

// Helper functions for protobuf pointer handling
func getStringPtr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func getBoolPtr(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func getUint32Ptr(ptr *uint32) uint32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func getUint64Ptr(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
