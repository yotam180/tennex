-- User Integrations table queries
-- Multi-platform integration management (WhatsApp, Email, Telegram, etc.)

-- name: UpsertUserIntegration :one
INSERT INTO user_integrations (
    user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen
) VALUES (
    $1::uuid, $2::text, $3::text, $4::text, $5::text, $6::text, $7::jsonb, $8::timestamptz
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
WHERE user_id = $1::uuid AND integration_type = $2::text;

-- name: GetUserIntegrationByExternalID :one
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE integration_type = $1::text AND external_id = $2::text;

-- name: UpdateUserIntegrationStatus :exec
UPDATE user_integrations 
SET status = $3::text, last_seen = $4::timestamptz, updated_at = NOW()
WHERE user_id = $1::uuid AND integration_type = $2::text;

-- name: ListUserIntegrations :many
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE user_id = $1::uuid
ORDER BY created_at DESC;

-- name: ListConnectedIntegrations :many
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE status = 'connected'
ORDER BY last_seen DESC;

-- name: ListIntegrationsByType :many
SELECT id, user_id, integration_type, external_id, status, display_name, avatar_url, metadata, last_seen, created_at, updated_at
FROM user_integrations 
WHERE integration_type = $1::text AND status = 'connected'
ORDER BY last_seen DESC;

-- name: DeleteUserIntegration :exec
DELETE FROM user_integrations 
WHERE user_id = $1::uuid AND integration_type = $2::text;
