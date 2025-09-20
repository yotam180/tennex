package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Handler processes WhatsApp events and converts them to our internal format
type Handler struct {
	logger    *zap.Logger
	publisher Publisher
	accountID string
	deviceID  string
}

// Publisher defines the interface for publishing events
type Publisher interface {
	PublishEvent(ctx context.Context, event *Event) error
}

// Event represents an internal event in our system
type Event struct {
	ID            string                 `json:"id"`
	Sequence      int64                  `json:"seq,omitempty"` // Set by event store
	Timestamp     time.Time              `json:"ts"`
	Type          EventType              `json:"type"`
	ConvoID       string                 `json:"convo_id"`
	WAMessageID   *string                `json:"wa_message_id,omitempty"`
	SenderJID     string                 `json:"sender_jid,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	AttachmentRef *string                `json:"attachment_ref,omitempty"`
	DeviceID      string                 `json:"device_id"`
	AccountID     string                 `json:"account_id"`
}

// EventType represents the type of event
type EventType string

const (
	EventTypeMessageIn  EventType = "msg_in"
	EventTypeMessageOut EventType = "msg_out"
	EventTypeEdit       EventType = "edit"
	EventTypeDelete     EventType = "delete"
	EventTypeReaction   EventType = "reaction"
	EventTypeRead       EventType = "read"
	EventTypeDelivery   EventType = "delivery"
	EventTypeContact    EventType = "contact"
	EventTypeThreadMeta EventType = "thread_meta"
	EventTypeMediaKey   EventType = "media_key"
	EventTypePresence   EventType = "presence"
	EventTypeConnection EventType = "connection"
)

// NewHandler creates a new event handler
func NewHandler(logger *zap.Logger, publisher Publisher, accountID, deviceID string) *Handler {
	return &Handler{
		logger:    logger,
		publisher: publisher,
		accountID: accountID,
		deviceID:  deviceID,
	}
}

// HandleMessage processes incoming WhatsApp messages
func (h *Handler) HandleMessage(msg *events.Message) {
	ctx := context.Background()

	// Determine conversation ID (group or individual)
	convoID := msg.Info.Chat.String()

	event := &Event{
		ID:          uuid.New().String(),
		Timestamp:   msg.Info.Timestamp,
		Type:        EventTypeMessageIn,
		ConvoID:     convoID,
		WAMessageID: &msg.Info.ID,
		SenderJID:   msg.Info.Sender.String(),
		DeviceID:    h.deviceID,
		AccountID:   h.accountID,
		Payload:     h.extractMessagePayload(msg),
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish message event",
			zap.Error(err),
			zap.String("message_id", msg.Info.ID),
			zap.String("from", msg.Info.Sender.String()))
	} else {
		h.logger.Debug("Published message event",
			zap.String("event_id", event.ID),
			zap.String("message_id", msg.Info.ID))
	}
}

// HandleReceipt processes message receipts (delivery, read, etc.)
func (h *Handler) HandleReceipt(receipt *events.Receipt) {
	ctx := context.Background()

	for _, msgID := range receipt.MessageIDs {
		event := &Event{
			ID:          uuid.New().String(),
			Timestamp:   receipt.Timestamp,
			Type:        EventTypeDelivery,
			ConvoID:     receipt.Chat.String(),
			WAMessageID: &msgID,
			SenderJID:   receipt.Sender.String(),
			DeviceID:    h.deviceID,
			AccountID:   h.accountID,
			Payload: map[string]interface{}{
				"receipt_type": string(receipt.Type),
				"participant":  receipt.Sender.String(),
			},
		}

		if err := h.publisher.PublishEvent(ctx, event); err != nil {
			h.logger.Error("Failed to publish receipt event",
				zap.Error(err),
				zap.String("message_id", msgID),
				zap.String("type", string(receipt.Type)))
		}
	}
}

// HandlePresence processes presence updates
func (h *Handler) HandlePresence(presence *events.Presence) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypePresence,
		ConvoID:   presence.From.String(),
		SenderJID: presence.From.String(),
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"presence":  "available", // Simplified for now, can be enhanced
			"last_seen": time.Now(),
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish presence event",
			zap.Error(err),
			zap.String("from", presence.From.String()))
	}
}

// HandleContact processes contact updates
func (h *Handler) HandleContact(contact *events.Contact) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypeContact,
		ConvoID:   contact.JID.String(),
		SenderJID: contact.JID.String(),
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"jid": contact.JID.String(),
			// Note: Contact event structure has changed in newer whatsmeow versions
			// Additional fields can be added based on the actual Contact struct
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish contact event",
			zap.Error(err),
			zap.String("jid", contact.JID.String()))
	}
}

// HandlePushName processes push name updates
func (h *Handler) HandlePushName(pushName *events.PushName) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypeContact,
		ConvoID:   pushName.JID.String(),
		SenderJID: pushName.JID.String(),
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"push_name": pushName.NewPushName,
			"action":    "push_name_update",
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish push name event",
			zap.Error(err),
			zap.String("jid", pushName.JID.String()))
	}
}

// HandleGroupInfo processes group information updates
func (h *Handler) HandleGroupInfo(groupInfo *events.GroupInfo) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: groupInfo.Timestamp,
		Type:      EventTypeThreadMeta,
		ConvoID:   groupInfo.JID.String(),
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"jid":  groupInfo.JID.String(),
			"name": groupInfo.Name,
			// Note: GroupInfo structure has changed in newer whatsmeow versions
			// Additional fields can be added based on the actual GroupInfo struct
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish group info event",
			zap.Error(err),
			zap.String("group_jid", groupInfo.JID.String()))
	}
}

// HandleLoggedOut processes logout events
func (h *Handler) HandleLoggedOut(loggedOut *events.LoggedOut) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypeConnection,
		ConvoID:   "system",
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"action": "logged_out",
			"reason": string(loggedOut.Reason),
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish logged out event", zap.Error(err))
	}
}

// HandleConnected processes connection events
func (h *Handler) HandleConnected(connected *events.Connected) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypeConnection,
		ConvoID:   "system",
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"action": "connected",
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish connected event", zap.Error(err))
	}
}

// HandleDisconnected processes disconnection events
func (h *Handler) HandleDisconnected(disconnected *events.Disconnected) {
	ctx := context.Background()

	event := &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypeConnection,
		ConvoID:   "system",
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"action": "disconnected",
		},
	}

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		h.logger.Error("Failed to publish disconnected event", zap.Error(err))
	}
}

// extractMessagePayload extracts relevant data from a WhatsApp message
func (h *Handler) extractMessagePayload(msg *events.Message) map[string]interface{} {
	payload := map[string]interface{}{
		"from_me":      msg.Info.IsFromMe,
		"broadcast":    msg.Info.IsGroup,
		"message_type": "text", // Default, will be overridden based on content
	}

	// Extract message content
	if msg.Message.GetConversation() != "" {
		payload["body"] = msg.Message.GetConversation()
	} else if extendedText := msg.Message.GetExtendedTextMessage(); extendedText != nil {
		payload["body"] = extendedText.GetText()
		payload["message_type"] = "extended_text"
	} else if imageMsg := msg.Message.GetImageMessage(); imageMsg != nil {
		payload["body"] = imageMsg.GetCaption()
		payload["message_type"] = "image"
		payload["media"] = map[string]interface{}{
			"mime_type": imageMsg.GetMimetype(),
			"sha256":    imageMsg.GetFileSHA256(),
			"size":      imageMsg.GetFileLength(),
		}
	} else if audioMsg := msg.Message.GetAudioMessage(); audioMsg != nil {
		payload["message_type"] = "audio"
		payload["media"] = map[string]interface{}{
			"mime_type": audioMsg.GetMimetype(),
			"sha256":    audioMsg.GetFileSHA256(),
			"size":      audioMsg.GetFileLength(),
			"duration":  audioMsg.GetSeconds(),
		}
	} else if videoMsg := msg.Message.GetVideoMessage(); videoMsg != nil {
		payload["body"] = videoMsg.GetCaption()
		payload["message_type"] = "video"
		payload["media"] = map[string]interface{}{
			"mime_type": videoMsg.GetMimetype(),
			"sha256":    videoMsg.GetFileSHA256(),
			"size":      videoMsg.GetFileLength(),
			"duration":  videoMsg.GetSeconds(),
		}
	} else if documentMsg := msg.Message.GetDocumentMessage(); documentMsg != nil {
		payload["body"] = documentMsg.GetFileName()
		payload["message_type"] = "document"
		payload["media"] = map[string]interface{}{
			"mime_type": documentMsg.GetMimetype(),
			"sha256":    documentMsg.GetFileSHA256(),
			"size":      documentMsg.GetFileLength(),
			"filename":  documentMsg.GetFileName(),
		}
	}

	// Handle quoted messages
	if contextInfo := msg.Message.GetExtendedTextMessage().GetContextInfo(); contextInfo != nil {
		if quotedMsg := contextInfo.GetQuotedMessage(); quotedMsg != nil {
			payload["quoted_message"] = map[string]interface{}{
				"id":   contextInfo.GetStanzaId(),
				"from": contextInfo.GetParticipant(),
			}
		}
	}

	return payload
}

// extractParticipants extracts participant information from group info
// Note: Simplified for now due to API changes, can be enhanced later
func (h *Handler) extractParticipants(participants interface{}) []map[string]interface{} {
	// Simplified implementation - return empty slice for now
	// This can be enhanced once we understand the new participant structure
	return []map[string]interface{}{}
}

// CreateOutboundMessageEvent creates an event for an outbound message
func (h *Handler) CreateOutboundMessageEvent(convoID, clientMsgUUID, body string) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Type:      EventTypeMessageOut,
		ConvoID:   convoID,
		DeviceID:  h.deviceID,
		AccountID: h.accountID,
		Payload: map[string]interface{}{
			"body":            body,
			"client_msg_uuid": clientMsgUUID,
			"status":          "queued",
			"message_type":    "text",
		},
	}
}
