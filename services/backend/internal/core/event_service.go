package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/repo"
	"github.com/tennex/pkg/events"
)

// EventService handles event business logic
type EventService struct {
	eventRepo repo.EventRepository
	nats      *nats.Conn
	logger    *zap.Logger
}

// NewEventService creates a new event service
func NewEventService(eventRepo repo.EventRepository, natsConn *nats.Conn, logger *zap.Logger) *EventService {
	return &EventService{
		eventRepo: eventRepo,
		nats:      natsConn,
		logger:    logger.Named("event_service"),
	}
}

// PublishInbound publishes an inbound event from the bridge
func (s *EventService) PublishInbound(ctx context.Context, event *repo.Event) (int64, bool, error) {
	s.logger.Debug("Publishing inbound event",
		zap.String("event_id", event.ID.String()),
		zap.String("type", event.Type),
		zap.String("account_id", event.AccountID))

	// Insert event (idempotent)
	result, err := s.eventRepo.InsertEvent(ctx, repo.InsertEventParams{
		ID:            event.ID,
		Type:          event.Type,
		AccountID:     event.AccountID,
		DeviceID:      event.DeviceID,
		ConvoID:       event.ConvoID,
		WaMessageID:   event.WaMessageID,
		SenderJid:     event.SenderJid,
		Payload:       event.Payload,
		AttachmentRef: event.AttachmentRef,
	})
	if err != nil {
		s.logger.Error("Failed to insert event", zap.Error(err))
		return 0, false, fmt.Errorf("failed to insert event: %w", err)
	}

	created := result.Seq != 0 // seq is 0 if event already existed

	if created {
		// Publish notification to NATS
		if err := s.publishNotification(event.AccountID, result.Seq); err != nil {
			s.logger.Warn("Failed to publish notification", zap.Error(err))
			// Don't fail the request if notification fails
		}

		s.logger.Info("Event published",
			zap.String("event_id", event.ID.String()),
			zap.Int64("seq", result.Seq),
			zap.String("account_id", event.AccountID))
	} else {
		s.logger.Debug("Event already exists (idempotent)",
			zap.String("event_id", event.ID.String()))
	}

	return result.Seq, created, nil
}

// GetEventsSince retrieves events for an account since a sequence number
func (s *EventService) GetEventsSince(ctx context.Context, accountID string, since int64, limit int32) ([]repo.Event, error) {
	s.logger.Debug("Getting events since",
		zap.String("account_id", accountID),
		zap.Int64("since", since),
		zap.Int32("limit", limit))

	events, err := s.eventRepo.GetEventsSince(ctx, repo.GetEventsSinceParams{
		AccountID: accountID,
		Seq:       since,
		Limit:     limit,
	})
	if err != nil {
		s.logger.Error("Failed to get events", zap.Error(err))
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	s.logger.Debug("Retrieved events",
		zap.String("account_id", accountID),
		zap.Int("count", len(events)))

	return events, nil
}

// GetLatestEventSeq gets the latest sequence number for an account
func (s *EventService) GetLatestEventSeq(ctx context.Context, accountID string) (int64, error) {
	seq, err := s.eventRepo.GetLatestEventSeq(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest event seq: %w", err)
	}
	return seq, nil
}

// publishNotification publishes an ephemeral notification about new events
func (s *EventService) publishNotification(accountID string, nextSeq int64) error {
	subject := fmt.Sprintf("notify.account.%s", accountID)

	notification := map[string]interface{}{
		"account_id": accountID,
		"next_seq":   nextSeq,
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	return s.nats.Publish(subject, data)
}

// CreateMessageOutEvent creates a pending outbound message event
func (s *EventService) CreateMessageOutEvent(ctx context.Context, accountID, convoID, clientMsgUUID string, payload json.RawMessage) (int64, error) {
	event := &repo.Event{
		ID:        uuid.New(),
		Type:      events.TypeMessageOutPending,
		AccountID: accountID,
		ConvoID:   convoID,
		Payload:   payload,
	}

	seq, _, err := s.PublishInbound(ctx, event)
	if err != nil {
		return 0, fmt.Errorf("failed to create message out event: %w", err)
	}

	return seq, nil
}
