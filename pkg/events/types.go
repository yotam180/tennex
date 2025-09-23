// Package events defines event types and constants for the Tennex system
package events

import "time"

// Event types that can occur in the system
const (
	// Inbound message events
	TypeMessageIn = "msg_in"

	// Outbound message events
	TypeMessageOutPending = "msg_out_pending" // Queued for sending
	TypeMessageOutSent    = "msg_out_sent"    // Successfully sent
	TypeMessageDelivery   = "msg_delivery"    // Delivery receipt

	// Presence and status events
	TypePresence      = "presence"       // User online/offline status
	TypeContactUpdate = "contact_update" // Contact info changed

	// History synchronization
	TypeHistorySync = "history_sync" // WhatsApp history import
)

// Message content types
const (
	ContentTypeText     = "text"
	ContentTypeImage    = "image"
	ContentTypeAudio    = "audio"
	ContentTypeVideo    = "video"
	ContentTypeDocument = "document"
	ContentTypeSticker  = "sticker"
	ContentTypeLocation = "location"
	ContentTypeContact  = "contact"
)

// Account status values
const (
	AccountStatusDisconnected = "disconnected"
	AccountStatusConnecting   = "connecting"
	AccountStatusConnected    = "connected"
	AccountStatusError        = "error"
)

// Outbox status values
const (
	OutboxStatusQueued  = "queued"
	OutboxStatusSending = "sending"
	OutboxStatusSent    = "sent"
	OutboxStatusFailed  = "failed"
	OutboxStatusRetry   = "retry"
)

// MessageInPayload represents the payload for inbound messages
type MessageInPayload struct {
	ContentType   string                 `json:"content_type"`
	Content       map[string]interface{} `json:"content"`
	IsFromMe      bool                   `json:"is_from_me"`
	SenderName    string                 `json:"sender_name,omitempty"`
	ReplyTo       string                 `json:"reply_to,omitempty"`
	ForwardedFrom string                 `json:"forwarded_from,omitempty"`
	EditedAt      *time.Time             `json:"edited_at,omitempty"`
	DeletedAt     *time.Time             `json:"deleted_at,omitempty"`
}

// MessageOutPayload represents the payload for outbound messages
type MessageOutPayload struct {
	ContentType      string                 `json:"content_type"`
	Content          map[string]interface{} `json:"content"`
	ToJID            string                 `json:"to_jid"`
	ReplyToMessageID string                 `json:"reply_to_message_id,omitempty"`
	ClientMsgUUID    string                 `json:"client_msg_uuid"`
}

// DeliveryPayload represents message delivery status
type DeliveryPayload struct {
	WAMessageID   string     `json:"wa_message_id"`
	Status        string     `json:"status"` // "delivered", "read", "failed"
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	ClientMsgUUID string     `json:"client_msg_uuid,omitempty"`
}

// PresencePayload represents user presence information
type PresencePayload struct {
	JID         string     `json:"jid"`
	IsOnline    bool       `json:"is_online"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	IsTyping    bool       `json:"is_typing,omitempty"`
	IsRecording bool       `json:"is_recording,omitempty"`
}

// ContactUpdatePayload represents contact information changes
type ContactUpdatePayload struct {
	JID         string `json:"jid"`
	DisplayName string `json:"display_name,omitempty"`
	PushName    string `json:"push_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	IsBlocked   bool   `json:"is_blocked"`
}

// HistorySyncPayload represents history synchronization metadata
type HistorySyncPayload struct {
	ConversationCount int        `json:"conversation_count"`
	MessageCount      int        `json:"message_count"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           time.Time  `json:"end_time"`
	SyncType          string     `json:"sync_type"` // "initial", "incremental"
	Progress          float64    `json:"progress"`  // 0.0 to 1.0
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// AttachmentRef represents a reference to media content
type AttachmentRef struct {
	ContentHash string `json:"content_hash"`
	MimeType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	StorageURL  string `json:"storage_url"`
	Filename    string `json:"filename,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Duration    int    `json:"duration,omitempty"` // seconds for audio/video
}
