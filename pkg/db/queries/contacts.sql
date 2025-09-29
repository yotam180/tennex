-- Contacts table queries
-- Integration-specific contact management
-- name: UpsertContact :one
INSERT INTO contacts (
        user_integration_id,
        external_contact_id,
        integration_type,
        display_name,
        first_name,
        last_name,
        phone_number,
        username,
        is_blocked,
        is_favorite,
        last_seen,
        avatar_url,
        platform_metadata
    )
VALUES (
        @user_integration_id::int,
        @external_contact_id::text,
        @integration_type::text,
        @display_name::text,
        @first_name::text,
        @last_name::text,
        @phone_number::text,
        @username::text,
        @is_blocked::bool,
        @is_favorite::bool,
        @last_seen::timestamptz,
        @avatar_url::text,
        @platform_metadata::jsonb
    ) ON CONFLICT (user_integration_id, external_contact_id) DO
UPDATE
SET display_name = EXCLUDED.display_name,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    phone_number = EXCLUDED.phone_number,
    username = EXCLUDED.username,
    is_blocked = EXCLUDED.is_blocked,
    is_favorite = EXCLUDED.is_favorite,
    last_seen = EXCLUDED.last_seen,
    avatar_url = EXCLUDED.avatar_url,
    platform_metadata = EXCLUDED.platform_metadata,
    updated_at = NOW()
RETURNING id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at;
-- name: GetContactByExternalID :one
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: GetContactByID :one
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE id = $1::uuid;
-- name: ListUserIntegrationContacts :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
ORDER BY display_name ASC NULLS LAST;
-- name: ListFavoriteContacts :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND is_favorite = true
ORDER BY display_name ASC NULLS LAST;
-- name: ListBlockedContacts :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND is_blocked = true
ORDER BY display_name ASC NULLS LAST;
-- name: SearchContacts :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND (
        display_name ILIKE '%' || $2::text || '%'
        OR first_name ILIKE '%' || $2::text || '%'
        OR last_name ILIKE '%' || $2::text || '%'
        OR username ILIKE '%' || $2::text || '%'
        OR phone_number ILIKE '%' || $2::text || '%'
    )
ORDER BY display_name ASC NULLS LAST
LIMIT $3::int;
-- name: SearchContactsByPhone :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND phone_number = $2::text;
-- name: GetOnlineContacts :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND last_seen > NOW() - INTERVAL '5 minutes'
    AND is_blocked = false
ORDER BY last_seen DESC;
-- name: GetRecentlyActiveContacts :many
SELECT id,
    user_integration_id,
    external_contact_id,
    integration_type,
    display_name,
    first_name,
    last_name,
    phone_number,
    username,
    is_blocked,
    is_favorite,
    last_seen,
    avatar_url,
    platform_metadata,
    created_at,
    updated_at
FROM contacts
WHERE user_integration_id = $1::int
    AND last_seen > NOW() - INTERVAL '24 hours'
    AND is_blocked = false
ORDER BY last_seen DESC
LIMIT $2::int;
-- name: UpdateContactInfo :exec
UPDATE contacts
SET display_name = $3::text,
    first_name = $4::text,
    last_name = $5::text,
    phone_number = $6::text,
    username = $7::text,
    avatar_url = $8::text,
    platform_metadata = $9::jsonb,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: UpdateContactLastSeen :exec
UPDATE contacts
SET last_seen = $3::timestamptz,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: BlockContact :exec
UPDATE contacts
SET is_blocked = true,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: UnblockContact :exec
UPDATE contacts
SET is_blocked = false,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: AddToFavorites :exec
UPDATE contacts
SET is_favorite = true,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: RemoveFromFavorites :exec
UPDATE contacts
SET is_favorite = false,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: DeleteContact :exec
DELETE FROM contacts
WHERE user_integration_id = $1::int
    AND external_contact_id = $2::text;
-- name: CountContacts :one
SELECT COUNT(*)
FROM contacts
WHERE user_integration_id = $1::int;
-- name: CountFavoriteContacts :one
SELECT COUNT(*)
FROM contacts
WHERE user_integration_id = $1::int
    AND is_favorite = true;
-- name: CountBlockedContacts :one
SELECT COUNT(*)
FROM contacts
WHERE user_integration_id = $1::int
    AND is_blocked = true;
-- name: GetContactStats :one
SELECT COUNT(*) as total_contacts,
    COUNT(*) FILTER (
        WHERE is_favorite = true
    ) as favorite_contacts,
    COUNT(*) FILTER (
        WHERE is_blocked = true
    ) as blocked_contacts,
    COUNT(*) FILTER (
        WHERE last_seen > NOW() - INTERVAL '24 hours'
    ) as recently_active
FROM contacts
WHERE user_integration_id = $1::int;