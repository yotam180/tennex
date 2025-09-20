package publisher

import (
	"context"
	"encoding/json"

	"github.com/tennex/bridge/internal/events"
	"go.uber.org/zap"
)

// ConsolePublisher is a simple publisher that logs events to console
// This is useful for development and debugging
type ConsolePublisher struct {
	logger *zap.Logger
}

// NewConsolePublisher creates a new console publisher
func NewConsolePublisher(logger *zap.Logger) *ConsolePublisher {
	return &ConsolePublisher{
		logger: logger,
	}
}

// PublishEvent logs the event to console
func (p *ConsolePublisher) PublishEvent(ctx context.Context, event *events.Event) error {
	// Pretty print the event as JSON
	eventJSON, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		p.logger.Error("Failed to marshal event to JSON",
			zap.Error(err),
			zap.String("event_id", event.ID))
		return err
	}

	// Log the event with structured fields
	p.logger.Info("Publishing event",
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)),
		zap.String("convo_id", event.ConvoID),
		zap.Time("timestamp", event.Timestamp),
		zap.String("sender_jid", event.SenderJID),
		zap.String("account_id", event.AccountID),
		zap.String("device_id", event.DeviceID))

	// Also log the full event JSON for debugging
	p.logger.Debug("Event payload", zap.String("json", string(eventJSON)))

	return nil
}
