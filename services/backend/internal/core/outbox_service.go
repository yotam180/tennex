package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/repo"
	"github.com/tennex/pkg/events"
)

// OutboxService handles outbound message queue
type OutboxService struct {
	outboxRepo repo.OutboxRepository
	eventRepo  repo.EventRepository
	logger     *zap.Logger
}

// NewOutboxService creates a new outbox service
func NewOutboxService(outboxRepo repo.OutboxRepository, eventRepo repo.EventRepository, logger *zap.Logger) *OutboxService {
	return &OutboxService{
		outboxRepo: outboxRepo,
		eventRepo:  eventRepo,
		logger:     logger.Named("outbox_service"),
	}
}

// CreateOutboxEntry creates a new outbox entry with transactional guarantees
func (s *OutboxService) CreateOutboxEntry(ctx context.Context, clientMsgUUID uuid.UUID, accountID, convoID string, serverMsgID int64) error {
	s.logger.Debug("Creating outbox entry",
		zap.String("client_msg_uuid", clientMsgUUID.String()),
		zap.String("account_id", accountID),
		zap.Int64("server_msg_id", serverMsgID))

	_, err := s.outboxRepo.CreateOutboxEntry(ctx, repo.CreateOutboxEntryParams{
		ClientMsgUuid: clientMsgUUID,
		AccountID:     accountID,
		ConvoID:       convoID,
		ServerMsgID:   sql.NullInt64{Int64: serverMsgID, Valid: true},
		Status:        events.OutboxStatusQueued,
	})
	if err != nil {
		s.logger.Error("Failed to create outbox entry", zap.Error(err))
		return fmt.Errorf("failed to create outbox entry: %w", err)
	}

	s.logger.Info("Outbox entry created",
		zap.String("client_msg_uuid", clientMsgUUID.String()),
		zap.String("status", events.OutboxStatusQueued))

	return nil
}

// GetPendingEntries retrieves pending outbox entries for processing
func (s *OutboxService) GetPendingEntries(ctx context.Context, limit int32) ([]repo.Outbox, error) {
	entries, err := s.outboxRepo.GetPendingOutboxEntries(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox entries: %w", err)
	}

	s.logger.Debug("Retrieved pending outbox entries", zap.Int("count", len(entries)))
	return entries, nil
}

// UpdateEntryStatus updates the status of an outbox entry
func (s *OutboxService) UpdateEntryStatus(ctx context.Context, clientMsgUUID uuid.UUID, status string, errorMsg string) error {
	s.logger.Debug("Updating outbox entry status",
		zap.String("client_msg_uuid", clientMsgUUID.String()),
		zap.String("status", status),
		zap.String("error", errorMsg))

	var lastError sql.NullString
	if errorMsg != "" {
		lastError = sql.NullString{String: errorMsg, Valid: true}
	}

	err := s.outboxRepo.UpdateOutboxStatus(ctx, repo.UpdateOutboxStatusParams{
		ClientMsgUuid: clientMsgUUID,
		Status:        status,
		LastError:     lastError,
	})
	if err != nil {
		s.logger.Error("Failed to update outbox status", zap.Error(err))
		return fmt.Errorf("failed to update outbox status: %w", err)
	}

	return nil
}

// GetEntry retrieves a specific outbox entry
func (s *OutboxService) GetEntry(ctx context.Context, clientMsgUUID uuid.UUID) (*repo.Outbox, error) {
	entry, err := s.outboxRepo.GetOutboxEntry(ctx, clientMsgUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox entry: %w", err)
	}
	return &entry, nil
}

// OutboxWorker processes outbox entries
type OutboxWorker struct {
	outboxService *OutboxService
	logger        *zap.Logger
	stopCh        chan struct{}
}

// NewOutboxWorker creates a new outbox worker
func NewOutboxWorker(outboxService *OutboxService, logger *zap.Logger) *OutboxWorker {
	return &OutboxWorker{
		outboxService: outboxService,
		logger:        logger.Named("outbox_worker"),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the outbox worker
func (w *OutboxWorker) Start(ctx context.Context) {
	w.logger.Info("Starting outbox worker")
	defer w.logger.Info("Outbox worker stopped")

	ticker := time.NewTicker(5 * time.Second) // Process every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processOutboxEntries(ctx)
		}
	}
}

// Stop stops the outbox worker
func (w *OutboxWorker) Stop() {
	close(w.stopCh)
}

// processOutboxEntries processes pending outbox entries
func (w *OutboxWorker) processOutboxEntries(ctx context.Context) {
	entries, err := w.outboxService.GetPendingEntries(ctx, 50) // Process up to 50 at a time
	if err != nil {
		w.logger.Error("Failed to get pending entries", zap.Error(err))
		return
	}

	if len(entries) == 0 {
		return
	}

	w.logger.Debug("Processing outbox entries", zap.Int("count", len(entries)))

	for _, entry := range entries {
		if err := w.processEntry(ctx, entry); err != nil {
			w.logger.Error("Failed to process outbox entry",
				zap.String("client_msg_uuid", entry.ClientMsgUuid.String()),
				zap.Error(err))

			// Mark as failed
			if updateErr := w.outboxService.UpdateEntryStatus(ctx, entry.ClientMsgUuid, events.OutboxStatusFailed, err.Error()); updateErr != nil {
				w.logger.Error("Failed to mark entry as failed", zap.Error(updateErr))
			}
		}
	}
}

// processEntry processes a single outbox entry
func (w *OutboxWorker) processEntry(ctx context.Context, entry repo.Outbox) error {
	// Mark as sending
	if err := w.outboxService.UpdateEntryStatus(ctx, entry.ClientMsgUuid, events.OutboxStatusSending, ""); err != nil {
		return fmt.Errorf("failed to mark as sending: %w", err)
	}

	// TODO: Call bridge service to actually send the message
	// For now, just simulate success after a short delay
	time.Sleep(100 * time.Millisecond)

	// Mark as sent
	if err := w.outboxService.UpdateEntryStatus(ctx, entry.ClientMsgUuid, events.OutboxStatusSent, ""); err != nil {
		return fmt.Errorf("failed to mark as sent: %w", err)
	}

	w.logger.Info("Message sent successfully",
		zap.String("client_msg_uuid", entry.ClientMsgUuid.String()),
		zap.String("account_id", entry.AccountID))

	return nil
}
