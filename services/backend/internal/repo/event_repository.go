package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventRepository struct {
	db *pgxpool.Pool
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *pgxpool.Pool) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) InsertEvent(ctx context.Context, params InsertEventParams) (InsertEventResult, error) {
	query := `
		INSERT INTO events (id, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
		RETURNING seq, ts`

	var result InsertEventResult
	err := r.db.QueryRow(ctx, query,
		params.ID,
		params.Type,
		params.AccountID,
		params.DeviceID,
		params.ConvoID,
		params.WaMessageID,
		params.SenderJid,
		params.Payload,
		params.AttachmentRef,
	).Scan(&result.Seq, &result.Ts)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Conflict occurred, event already exists
			return InsertEventResult{Seq: 0}, nil
		}
		return InsertEventResult{}, fmt.Errorf("failed to insert event: %w", err)
	}

	return result, nil
}

func (r *eventRepository) GetEventsSince(ctx context.Context, params GetEventsSinceParams) ([]Event, error) {
	query := `
		SELECT seq, id, ts, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
		FROM events 
		WHERE account_id = $1 AND seq > $2
		ORDER BY seq ASC
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, params.AccountID, params.Seq, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(
			&event.Seq,
			&event.ID,
			&event.Ts,
			&event.Type,
			&event.AccountID,
			&event.DeviceID,
			&event.ConvoID,
			&event.WaMessageID,
			&event.SenderJid,
			&event.Payload,
			&event.AttachmentRef,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}

func (r *eventRepository) GetLatestEventSeq(ctx context.Context, accountID string) (int64, error) {
	query := `SELECT COALESCE(MAX(seq), 0) as latest_seq FROM events WHERE account_id = $1`

	var latestSeq int64
	err := r.db.QueryRow(ctx, query, accountID).Scan(&latestSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest event seq: %w", err)
	}

	return latestSeq, nil
}

func (r *eventRepository) GetEventByID(ctx context.Context, id uuid.UUID) (Event, error) {
	query := `
		SELECT seq, id, ts, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
		FROM events 
		WHERE id = $1`

	var event Event
	err := r.db.QueryRow(ctx, query, id).Scan(
		&event.Seq,
		&event.ID,
		&event.Ts,
		&event.Type,
		&event.AccountID,
		&event.DeviceID,
		&event.ConvoID,
		&event.WaMessageID,
		&event.SenderJid,
		&event.Payload,
		&event.AttachmentRef,
	)
	if err != nil {
		return Event{}, fmt.Errorf("failed to get event by ID: %w", err)
	}

	return event, nil
}

func (r *eventRepository) CountEventsByAccount(ctx context.Context, accountID string) (int64, error) {
	query := `SELECT COUNT(*) as total_events FROM events WHERE account_id = $1`

	var count int64
	err := r.db.QueryRow(ctx, query, accountID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}
