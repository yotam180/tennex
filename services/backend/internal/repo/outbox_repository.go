package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRepository struct {
	db *pgxpool.Pool
}

// NewOutboxRepository creates a new outbox repository
func NewOutboxRepository(db *pgxpool.Pool) OutboxRepository {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) CreateOutboxEntry(ctx context.Context, params CreateOutboxEntryParams) (CreateOutboxEntryResult, error) {
	query := `
		INSERT INTO outbox (client_msg_uuid, account_id, convo_id, server_msg_id, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (client_msg_uuid) DO NOTHING
		RETURNING client_msg_uuid, created_at`

	var result CreateOutboxEntryResult
	err := r.db.QueryRow(ctx, query,
		params.ClientMsgUuid,
		params.AccountID,
		params.ConvoID,
		params.ServerMsgID,
		params.Status,
	).Scan(&result.ClientMsgUuid, &result.CreatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Conflict occurred, entry already exists
			return CreateOutboxEntryResult{ClientMsgUuid: params.ClientMsgUuid}, nil
		}
		return CreateOutboxEntryResult{}, fmt.Errorf("failed to create outbox entry: %w", err)
	}

	return result, nil
}

func (r *outboxRepository) GetPendingOutboxEntries(ctx context.Context, limit int32) ([]Outbox, error) {
	query := `
		SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
		FROM outbox 
		WHERE status IN ('queued', 'retry')
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending outbox entries: %w", err)
	}
	defer rows.Close()

	var entries []Outbox
	for rows.Next() {
		var entry Outbox
		err := rows.Scan(
			&entry.ClientMsgUuid,
			&entry.AccountID,
			&entry.ConvoID,
			&entry.ServerMsgID,
			&entry.Status,
			&entry.LastError,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return entries, nil
}

func (r *outboxRepository) UpdateOutboxStatus(ctx context.Context, params UpdateOutboxStatusParams) error {
	query := `
		UPDATE outbox 
		SET status = $2, last_error = $3, updated_at = NOW()
		WHERE client_msg_uuid = $1`

	result, err := r.db.Exec(ctx, query,
		params.ClientMsgUuid,
		params.Status,
		params.LastError,
	)
	if err != nil {
		return fmt.Errorf("failed to update outbox status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("outbox entry not found: %s", params.ClientMsgUuid)
	}

	return nil
}

func (r *outboxRepository) GetOutboxEntry(ctx context.Context, clientMsgUuid uuid.UUID) (Outbox, error) {
	query := `
		SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
		FROM outbox 
		WHERE client_msg_uuid = $1`

	var entry Outbox
	err := r.db.QueryRow(ctx, query, clientMsgUuid).Scan(
		&entry.ClientMsgUuid,
		&entry.AccountID,
		&entry.ConvoID,
		&entry.ServerMsgID,
		&entry.Status,
		&entry.LastError,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		return Outbox{}, fmt.Errorf("failed to get outbox entry: %w", err)
	}

	return entry, nil
}

func (r *outboxRepository) GetFailedOutboxEntries(ctx context.Context) ([]Outbox, error) {
	query := `
		SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
		FROM outbox 
		WHERE status = 'failed' AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed outbox entries: %w", err)
	}
	defer rows.Close()

	var entries []Outbox
	for rows.Next() {
		var entry Outbox
		err := rows.Scan(
			&entry.ClientMsgUuid,
			&entry.AccountID,
			&entry.ConvoID,
			&entry.ServerMsgID,
			&entry.Status,
			&entry.LastError,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return entries, nil
}

func (r *outboxRepository) RetryOutboxEntry(ctx context.Context, clientMsgUuid uuid.UUID) error {
	query := `
		UPDATE outbox 
		SET status = 'retry', last_error = NULL, updated_at = NOW()
		WHERE client_msg_uuid = $1 AND status = 'failed'`

	result, err := r.db.Exec(ctx, query, clientMsgUuid)
	if err != nil {
		return fmt.Errorf("failed to retry outbox entry: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("outbox entry not found or not in failed status: %s", clientMsgUuid)
	}

	return nil
}
