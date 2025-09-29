package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/repo"
	"github.com/tennex/pkg/events"
)

const (
	IntegrationTypeWhatsApp = "whatsapp"
	IntegrationTypeEmail    = "email"
	IntegrationTypeTelegram = "telegram"
	IntegrationTypeDiscord  = "discord"
	IntegrationTypeSlack    = "slack"
)

// IntegrationService handles user integration business logic
type IntegrationService struct {
	integrationRepo repo.IntegrationRepository
	logger          *zap.Logger
}

// NewIntegrationService creates a new integration service
func NewIntegrationService(integrationRepo repo.IntegrationRepository, logger *zap.Logger) *IntegrationService {
	return &IntegrationService{
		integrationRepo: integrationRepo,
		logger:          logger.Named("integration_service"),
	}
}

// UpsertUserIntegration creates or updates a user's integration
func (s *IntegrationService) UpsertUserIntegration(ctx context.Context, userID uuid.UUID, integrationType, externalID, displayName, avatarUrl, status string, metadata map[string]interface{}, lastSeen *time.Time) (*repo.UserIntegration, error) {
	s.logger.Debug("Upserting user integration",
		zap.String("user_id", userID.String()),
		zap.String("integration_type", integrationType),
		zap.String("external_id", externalID),
		zap.String("status", status))

	var displayNameNull sql.NullString
	if displayName != "" {
		displayNameNull = sql.NullString{String: displayName, Valid: true}
	}

	var avatarUrlNull sql.NullString
	if avatarUrl != "" {
		avatarUrlNull = sql.NullString{String: avatarUrl, Valid: true}
	}

	var metadataJSON json.RawMessage
	if len(metadata) > 0 {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = data
	} else {
		metadataJSON = json.RawMessage("{}")
	}

	var lastSeenNull sql.NullTime
	if lastSeen != nil {
		lastSeenNull = sql.NullTime{Time: *lastSeen, Valid: true}
	}

	integration, err := s.integrationRepo.UpsertUserIntegration(ctx, repo.UpsertUserIntegrationParams{
		UserID:          userID,
		IntegrationType: integrationType,
		ExternalID:      externalID,
		Status:          status,
		DisplayName:     displayNameNull,
		AvatarUrl:       avatarUrlNull,
		Metadata:        metadataJSON,
		LastSeen:        lastSeenNull,
	})
	if err != nil {
		s.logger.Error("Failed to upsert user integration", zap.Error(err))
		return nil, fmt.Errorf("failed to upsert user integration: %w", err)
	}

	s.logger.Info("User integration upserted",
		zap.String("user_id", userID.String()),
		zap.String("integration_type", integrationType),
		zap.String("external_id", externalID),
		zap.String("status", integration.Status))

	return &integration, nil
}

// GetUserIntegration retrieves a specific integration for a user
func (s *IntegrationService) GetUserIntegration(ctx context.Context, userID uuid.UUID, integrationType string) (*repo.UserIntegration, error) {
	integration, err := s.integrationRepo.GetUserIntegration(ctx, userID, integrationType)
	if err != nil {
		return nil, fmt.Errorf("failed to get user integration: %w", err)
	}
	return &integration, nil
}

// ListUserIntegrations retrieves all integrations for a user
func (s *IntegrationService) ListUserIntegrations(ctx context.Context, userID uuid.UUID) ([]repo.UserIntegration, error) {
	integrations, err := s.integrationRepo.ListUserIntegrations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user integrations: %w", err)
	}

	s.logger.Debug("Retrieved user integrations", zap.String("user_id", userID.String()), zap.Int("count", len(integrations)))
	return integrations, nil
}

// WhatsApp-specific helper methods for backward compatibility

// UpsertWhatsAppIntegration creates or updates a WhatsApp integration
func (s *IntegrationService) UpsertWhatsAppIntegration(ctx context.Context, userID uuid.UUID, waJid, displayName, avatarUrl string) (*repo.UserIntegration, error) {
	now := time.Now()
	return s.UpsertUserIntegration(ctx, userID, IntegrationTypeWhatsApp, waJid, displayName, avatarUrl, events.AccountStatusConnected, nil, &now)
}

// GetWhatsAppIntegration retrieves a user's WhatsApp integration
func (s *IntegrationService) GetWhatsAppIntegration(ctx context.Context, userID uuid.UUID) (*repo.UserIntegration, error) {
	return s.GetUserIntegration(ctx, userID, IntegrationTypeWhatsApp)
}

// SetWhatsAppConnected marks a WhatsApp integration as connected
func (s *IntegrationService) SetWhatsAppConnected(ctx context.Context, userID uuid.UUID, waJid, displayName string) error {
	now := time.Now()
	_, err := s.UpsertWhatsAppIntegration(ctx, userID, waJid, displayName, "")
	if err != nil {
		return err
	}

	return s.UpdateIntegrationStatus(ctx, userID, IntegrationTypeWhatsApp, events.AccountStatusConnected, &now)
}

// SetWhatsAppDisconnected marks a WhatsApp integration as disconnected
func (s *IntegrationService) SetWhatsAppDisconnected(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return s.UpdateIntegrationStatus(ctx, userID, IntegrationTypeWhatsApp, events.AccountStatusDisconnected, &now)
}

// SetWhatsAppError marks a WhatsApp integration as having an error
func (s *IntegrationService) SetWhatsAppError(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return s.UpdateIntegrationStatus(ctx, userID, IntegrationTypeWhatsApp, events.AccountStatusError, &now)
}

// UpdateIntegrationStatus updates the status of a user integration
func (s *IntegrationService) UpdateIntegrationStatus(ctx context.Context, userID uuid.UUID, integrationType, status string, lastSeen *time.Time) error {
	var lastSeenNull sql.NullTime
	if lastSeen != nil {
		lastSeenNull = sql.NullTime{Time: *lastSeen, Valid: true}
	}

	err := s.integrationRepo.UpdateUserIntegrationStatus(ctx, userID, integrationType, status, lastSeenNull)
	if err != nil {
		s.logger.Error("Failed to update integration status",
			zap.String("user_id", userID.String()),
			zap.String("integration_type", integrationType),
			zap.String("status", status),
			zap.Error(err))
		return fmt.Errorf("failed to update integration status: %w", err)
	}

	s.logger.Info("Integration status updated",
		zap.String("user_id", userID.String()),
		zap.String("integration_type", integrationType),
		zap.String("status", status))

	return nil
}

// DeleteUserIntegration removes a user's integration
func (s *IntegrationService) DeleteUserIntegration(ctx context.Context, userID uuid.UUID, integrationType string) error {
	err := s.integrationRepo.DeleteUserIntegration(ctx, userID, integrationType)
	if err != nil {
		s.logger.Error("Failed to delete user integration",
			zap.String("user_id", userID.String()),
			zap.String("integration_type", integrationType),
			zap.Error(err))
		return fmt.Errorf("failed to delete user integration: %w", err)
	}

	s.logger.Info("User integration deleted",
		zap.String("user_id", userID.String()),
		zap.String("integration_type", integrationType))

	return nil
}
