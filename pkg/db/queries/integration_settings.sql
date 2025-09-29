-- Integration settings table queries
-- Per-integration configuration management
-- name: UpsertIntegrationSetting :one
INSERT INTO integration_settings (
        user_integration_id,
        setting_key,
        setting_value
    )
VALUES ($1::int, $2::text, $3::jsonb) ON CONFLICT (user_integration_id, setting_key) DO
UPDATE
SET setting_value = EXCLUDED.setting_value,
    updated_at = NOW()
RETURNING id,
    user_integration_id,
    setting_key,
    setting_value,
    created_at,
    updated_at;
-- name: GetIntegrationSetting :one
SELECT id,
    user_integration_id,
    setting_key,
    setting_value,
    created_at,
    updated_at
FROM integration_settings
WHERE user_integration_id = $1::int
    AND setting_key = $2::text;
-- name: GetIntegrationSettingByID :one
SELECT id,
    user_integration_id,
    setting_key,
    setting_value,
    created_at,
    updated_at
FROM integration_settings
WHERE id = $1::uuid;
-- name: ListIntegrationSettings :many
SELECT id,
    user_integration_id,
    setting_key,
    setting_value,
    created_at,
    updated_at
FROM integration_settings
WHERE user_integration_id = $1::int
ORDER BY setting_key ASC;
-- name: GetIntegrationSettingsMap :many
SELECT setting_key,
    setting_value
FROM integration_settings
WHERE user_integration_id = $1::int;
-- name: UpdateIntegrationSetting :exec
UPDATE integration_settings
SET setting_value = $3::jsonb,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND setting_key = $2::text;
-- name: DeleteIntegrationSetting :exec
DELETE FROM integration_settings
WHERE user_integration_id = $1::int
    AND setting_key = $2::text;
-- name: DeleteAllIntegrationSettings :exec
DELETE FROM integration_settings
WHERE user_integration_id = $1::int;
-- name: CountIntegrationSettings :one
SELECT COUNT(*)
FROM integration_settings
WHERE user_integration_id = $1::int;