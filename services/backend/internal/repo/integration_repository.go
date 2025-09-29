package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type integrationRepository struct {
	db *pgxpool.Pool
}

// NewIntegrationRepository creates a new integration repository
func NewIntegrationRepository(db *pgxpool.Pool) IntegrationRepository {
	return &integrationRepository{db: db}
}

// UserIntegration represents a user's integration with an external platform
type UserIntegration struct {
	ID              int32           `json:"id"`
	UserID          uuid.UUID       `json:"user_id"`
	IntegrationType string          `json:"integration_type"`
	ExternalID      string          `json:"external_id"`
	Status          string          `json:"status"`
	DisplayName     sql.NullString  `json:"display_name"`
	AvatarUrl       sql.NullString  `json:"avatar_url"`
	Metadata        json.RawMessage `json:"metadata"`
	LastSeen        sql.NullTime    `json:"last_seen"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// UpsertUserIntegrationParams holds parameters for upserting a user integration
type UpsertUserIntegrationParams struct {
	UserID          uuid.UUID
	IntegrationType string
	ExternalID      string
	Status          string
	DisplayName     sql.NullString
	AvatarUrl       sql.NullString
	Metadata        json.RawMessage
	LastSeen        sql.NullTime
}

func (r *integrationRepository) UpsertUserIntegration(ctx context.Context, params UpsertUserIntegrationParams) (UserIntegration, error) {
	query := `
		INSERT INTO user_integrations (
			user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) ON CONFLICT (user_id, integration_type) DO UPDATE SET
			external_id = EXCLUDED.external_id,
			status = EXCLUDED.status,
			display_name = EXCLUDED.display_name,
			avatar_url = EXCLUDED.avatar_url,
			metadata = EXCLUDED.metadata,
			last_seen = EXCLUDED.last_seen,
			updated_at = NOW()
		RETURNING id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at`

	var integration UserIntegration
	err := r.db.QueryRow(ctx, query,
		params.UserID,
		params.IntegrationType,
		params.ExternalID,
		params.Status,
		params.DisplayName,
		params.AvatarUrl,
		params.Metadata,
		params.LastSeen,
	).Scan(
		&integration.ID,
		&integration.UserID,
		&integration.IntegrationType,
		&integration.ExternalID,
		&integration.Status,
		&integration.DisplayName,
		&integration.AvatarUrl,
		&integration.Metadata,
		&integration.LastSeen,
		&integration.CreatedAt,
		&integration.UpdatedAt,
	)
	if err != nil {
		return UserIntegration{}, fmt.Errorf("failed to upsert user integration: %w", err)
	}

	return integration, nil
}

func (r *integrationRepository) GetUserIntegration(ctx context.Context, userID uuid.UUID, integrationType string) (UserIntegration, error) {
	query := `
		SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
		FROM user_integrations 
		WHERE user_id = $1 AND integration_type = $2`

	var integration UserIntegration
	err := r.db.QueryRow(ctx, query, userID, integrationType).Scan(
		&integration.ID,
		&integration.UserID,
		&integration.IntegrationType,
		&integration.ExternalID,
		&integration.Status,
		&integration.DisplayName,
		&integration.AvatarUrl,
		&integration.Metadata,
		&integration.LastSeen,
		&integration.CreatedAt,
		&integration.UpdatedAt,
	)
	if err != nil {
		return UserIntegration{}, fmt.Errorf("failed to get user integration: %w", err)
	}

	return integration, nil
}

func (r *integrationRepository) GetUserIntegrationByExternalID(ctx context.Context, integrationType, externalID string) (UserIntegration, error) {
	query := `
		SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
		FROM user_integrations 
		WHERE integration_type = $1 AND external_id = $2`

	var integration UserIntegration
	err := r.db.QueryRow(ctx, query, integrationType, externalID).Scan(
		&integration.ID,
		&integration.UserID,
		&integration.IntegrationType,
		&integration.ExternalID,
		&integration.Status,
		&integration.DisplayName,
		&integration.AvatarUrl,
		&integration.Metadata,
		&integration.LastSeen,
		&integration.CreatedAt,
		&integration.UpdatedAt,
	)
	if err != nil {
		return UserIntegration{}, fmt.Errorf("failed to get user integration by external ID: %w", err)
	}

	return integration, nil
}

func (r *integrationRepository) ListUserIntegrations(ctx context.Context, userID uuid.UUID) ([]UserIntegration, error) {
	query := `
		SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
		FROM user_integrations 
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user integrations: %w", err)
	}
	defer rows.Close()

	var integrations []UserIntegration
	for rows.Next() {
		var integration UserIntegration
		err := rows.Scan(
			&integration.ID,
			&integration.UserID,
			&integration.IntegrationType,
			&integration.ExternalID,
			&integration.Status,
			&integration.DisplayName,
			&integration.AvatarUrl,
			&integration.Metadata,
			&integration.LastSeen,
			&integration.CreatedAt,
			&integration.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan integration: %w", err)
		}
		integrations = append(integrations, integration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return integrations, nil
}

func (r *integrationRepository) UpdateUserIntegrationStatus(ctx context.Context, userID uuid.UUID, integrationType, status string, lastSeen sql.NullTime) error {
	query := `
		UPDATE user_integrations 
		SET status = $3, last_seen = $4, updated_at = NOW()
		WHERE user_id = $1 AND integration_type = $2`

	result, err := r.db.Exec(ctx, query, userID, integrationType, status, lastSeen)
	if err != nil {
		return fmt.Errorf("failed to update integration status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("integration not found for user %s and type %s", userID, integrationType)
	}

	return nil
}

func (r *integrationRepository) DeleteUserIntegration(ctx context.Context, userID uuid.UUID, integrationType string) error {
	query := `DELETE FROM user_integrations WHERE user_id = $1 AND integration_type = $2`

	result, err := r.db.Exec(ctx, query, userID, integrationType)
	if err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("integration not found for user %s and type %s", userID, integrationType)
	}

	return nil
}
