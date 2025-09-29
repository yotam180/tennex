-- Conversation participants table queries
-- Group and channel participant management
-- name: UpsertConversationParticipant :one
INSERT INTO conversation_participants (
        conversation_id,
        external_user_id,
        integration_type,
        display_name,
        role,
        is_active,
        joined_at,
        left_at,
        added_by_external_id,
        platform_metadata
    )
VALUES (
        @conversation_id::uuid,
        @external_user_id::text,
        @integration_type::text,
        @display_name::text,
        @role::text,
        @is_active::bool,
        @joined_at::timestamptz,
        @left_at::timestamptz,
        @added_by_external_id::text,
        @platform_metadata::jsonb
    ) ON CONFLICT (conversation_id, external_user_id) DO
UPDATE
SET display_name = EXCLUDED.display_name,
    role = EXCLUDED.role,
    is_active = EXCLUDED.is_active,
    left_at = EXCLUDED.left_at,
    added_by_external_id = EXCLUDED.added_by_external_id,
    platform_metadata = EXCLUDED.platform_metadata,
    updated_at = NOW()
RETURNING id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at;
-- name: GetConversationParticipant :one
SELECT id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text;
-- name: GetConversationParticipantByID :one
SELECT id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at
FROM conversation_participants
WHERE id = $1::uuid;
-- name: ListConversationParticipants :many
SELECT id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at
FROM conversation_participants
WHERE conversation_id = $1::uuid
ORDER BY joined_at ASC;
-- name: ListActiveConversationParticipants :many
SELECT id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND is_active = true
ORDER BY joined_at ASC;
-- name: ListConversationAdmins :many
SELECT id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND role IN ('admin', 'owner')
    AND is_active = true
ORDER BY joined_at ASC;
-- name: GetConversationOwner :one
SELECT id,
    conversation_id,
    external_user_id,
    integration_type,
    display_name,
    role,
    is_active,
    joined_at,
    left_at,
    added_by_external_id,
    platform_metadata,
    created_at,
    updated_at
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND role = 'owner'
LIMIT 1;
-- name: ListUserConversations :many
SELECT DISTINCT c.id,
    c.user_integration_id,
    c.external_conversation_id,
    c.integration_type,
    c.conversation_type,
    c.name,
    c.description,
    c.avatar_url,
    c.is_archived,
    c.is_pinned,
    c.is_muted,
    c.mute_until,
    c.is_read_only,
    c.is_locked,
    c.unread_count,
    c.unread_mention_count,
    c.total_message_count,
    c.last_message_at,
    c.last_activity_at,
    c.platform_metadata,
    c.created_at,
    c.updated_at
FROM conversations c
    INNER JOIN conversation_participants cp ON c.id = cp.conversation_id
WHERE cp.external_user_id = $1::text
    AND cp.is_active = true
ORDER BY c.last_activity_at DESC NULLS LAST;
-- name: UpdateParticipantRole :exec
UPDATE conversation_participants
SET role = $3::text,
    updated_at = NOW()
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text;
-- name: UpdateParticipantDisplayName :exec
UPDATE conversation_participants
SET display_name = $3::text,
    updated_at = NOW()
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text;
-- name: RemoveParticipant :exec
UPDATE conversation_participants
SET is_active = false,
    left_at = NOW(),
    updated_at = NOW()
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text;
-- name: AddParticipant :exec
UPDATE conversation_participants
SET is_active = true,
    left_at = NULL,
    updated_at = NOW()
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text;
-- name: DeleteParticipant :exec
DELETE FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text;
-- name: CountConversationParticipants :one
SELECT COUNT(*)
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND is_active = true;
-- name: CountConversationAdmins :one
SELECT COUNT(*)
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND role IN ('admin', 'owner')
    AND is_active = true;
-- name: IsParticipantActive :one
SELECT EXISTS (
        SELECT 1
        FROM conversation_participants
        WHERE conversation_id = $1::uuid
            AND external_user_id = $2::text
            AND is_active = true
    );
-- name: IsParticipantAdmin :one
SELECT EXISTS (
        SELECT 1
        FROM conversation_participants
        WHERE conversation_id = $1::uuid
            AND external_user_id = $2::text
            AND role IN ('admin', 'owner')
            AND is_active = true
    );
-- name: GetParticipantRole :one
SELECT role
FROM conversation_participants
WHERE conversation_id = $1::uuid
    AND external_user_id = $2::text
    AND is_active = true;