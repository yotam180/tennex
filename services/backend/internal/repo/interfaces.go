package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Generated types from sqlc (placeholder - will be replaced by actual generated types)

type Event struct {
	Seq           int64           `json:"seq"`
	ID            uuid.UUID       `json:"id"`
	Ts            time.Time       `json:"ts"`
	Type          string          `json:"type"`
	AccountID     string          `json:"account_id"`
	DeviceID      sql.NullString  `json:"device_id"`
	ConvoID       string          `json:"convo_id"`
	WaMessageID   sql.NullString  `json:"wa_message_id"`
	SenderJid     sql.NullString  `json:"sender_jid"`
	Payload       json.RawMessage `json:"payload"`
	AttachmentRef json.RawMessage `json:"attachment_ref"`
}

type Outbox struct {
	ClientMsgUuid uuid.UUID      `json:"client_msg_uuid"`
	AccountID     string         `json:"account_id"`
	ConvoID       string         `json:"convo_id"`
	ServerMsgID   sql.NullInt64  `json:"server_msg_id"`
	Status        string         `json:"status"`
	LastError     sql.NullString `json:"last_error"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Account struct {
	ID          string         `json:"id"`
	WaJid       sql.NullString `json:"wa_jid"`
	DisplayName sql.NullString `json:"display_name"`
	AvatarUrl   sql.NullString `json:"avatar_url"`
	Status      string         `json:"status"`
	LastSeen    sql.NullTime   `json:"last_seen"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type MediaBlob struct {
	ContentHash string    `json:"content_hash"`
	MimeType    string    `json:"mime_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StorageUrl  string    `json:"storage_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// Repository parameter types
type InsertEventParams struct {
	ID            uuid.UUID
	Type          string
	AccountID     string
	DeviceID      sql.NullString
	ConvoID       string
	WaMessageID   sql.NullString
	SenderJid     sql.NullString
	Payload       json.RawMessage
	AttachmentRef json.RawMessage
}

type InsertEventResult struct {
	Seq int64
	Ts  time.Time
}

type GetEventsSinceParams struct {
	AccountID string
	Seq       int64
	Limit     int32
}

type CreateOutboxEntryParams struct {
	ClientMsgUuid uuid.UUID
	AccountID     string
	ConvoID       string
	ServerMsgID   sql.NullInt64
	Status        string
}

type CreateOutboxEntryResult struct {
	ClientMsgUuid uuid.UUID
	CreatedAt     time.Time
}

type UpdateOutboxStatusParams struct {
	ClientMsgUuid uuid.UUID
	Status        string
	LastError     sql.NullString
}

type UpsertAccountParams struct {
	ID          string
	WaJid       sql.NullString
	DisplayName sql.NullString
	AvatarUrl   sql.NullString
	Status      string
	LastSeen    sql.NullTime
}

type UpdateAccountStatusParams struct {
	ID       string
	Status   string
	LastSeen sql.NullTime
}

type ListAccountsParams struct {
	Limit  int32
	Offset int32
}

// Repository interfaces
type EventRepository interface {
	InsertEvent(ctx context.Context, params InsertEventParams) (InsertEventResult, error)
	GetEventsSince(ctx context.Context, params GetEventsSinceParams) ([]Event, error)
	GetLatestEventSeq(ctx context.Context, accountID string) (int64, error)
	GetEventByID(ctx context.Context, id uuid.UUID) (Event, error)
	CountEventsByAccount(ctx context.Context, accountID string) (int64, error)
}

type OutboxRepository interface {
	CreateOutboxEntry(ctx context.Context, params CreateOutboxEntryParams) (CreateOutboxEntryResult, error)
	GetPendingOutboxEntries(ctx context.Context, limit int32) ([]Outbox, error)
	UpdateOutboxStatus(ctx context.Context, params UpdateOutboxStatusParams) error
	GetOutboxEntry(ctx context.Context, clientMsgUuid uuid.UUID) (Outbox, error)
	GetFailedOutboxEntries(ctx context.Context) ([]Outbox, error)
	RetryOutboxEntry(ctx context.Context, clientMsgUuid uuid.UUID) error
}

type AccountRepository interface {
	UpsertAccount(ctx context.Context, params UpsertAccountParams) (Account, error)
	GetAccount(ctx context.Context, id string) (Account, error)
	GetAccountByWAJID(ctx context.Context, waJid string) (Account, error)
	UpdateAccountStatus(ctx context.Context, params UpdateAccountStatusParams) error
	ListAccounts(ctx context.Context, params ListAccountsParams) ([]Account, error)
	GetConnectedAccounts(ctx context.Context) ([]Account, error)
}

type IntegrationRepository interface {
	UpsertUserIntegration(ctx context.Context, params UpsertUserIntegrationParams) (UserIntegration, error)
	GetUserIntegration(ctx context.Context, userID uuid.UUID, integrationType string) (UserIntegration, error)
	GetUserIntegrationByExternalID(ctx context.Context, integrationType, externalID string) (UserIntegration, error)
	ListUserIntegrations(ctx context.Context, userID uuid.UUID) ([]UserIntegration, error)
	UpdateUserIntegrationStatus(ctx context.Context, userID uuid.UUID, integrationType, status string, lastSeen sql.NullTime) error
	DeleteUserIntegration(ctx context.Context, userID uuid.UUID, integrationType string) error
}
