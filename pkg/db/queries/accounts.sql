-- User Integrations table queries
-- Multi-platform integration management (WhatsApp, Email, Telegram, etc.)

-- name: UpsertUserIntegration :one
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
RETURNING id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at;

-- name: GetUserIntegration :one
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE user_id = $1 AND integration_type = $2;

-- name: GetUserIntegrationByExternalID :one
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE integration_type = $1 AND external_id = $2;

-- name: UpdateUserIntegrationStatus :exec
UPDATE user_integrations 
SET status = $3, last_seen = $4, updated_at = NOW()
WHERE user_id = $1 AND integration_type = $2;

-- name: ListUserIntegrations :many
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListConnectedIntegrations :many
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE status = 'connected'
ORDER BY last_seen DESC;

-- name: ListIntegrationsByType :many
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE integration_type = $1 AND status = 'connected'
ORDER BY last_seen DESC;

-- name: DeleteUserIntegration :exec
DELETE FROM user_integrations 
WHERE user_id = $1 AND integration_type = $2;
