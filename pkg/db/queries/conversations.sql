-- Conversations table queries
-- Platform-agnostic conversation/chat/channel management
-- name: UpsertConversation :one
INSERT INTO conversations (
        user_integration_id,
        external_conversation_id,
        integration_type,
        conversation_type,
        name,
        description,
        avatar_url,
        is_archived,
        is_pinned,
        is_muted,
        mute_until,
        is_read_only,
        is_locked,
        unread_count,
        unread_mention_count,
        total_message_count,
        last_message_at,
        last_activity_at,
        platform_metadata
    )
VALUES (
        @user_integration_id::int,
        @external_conversation_id::text,
        @integration_type::text,
        @conversation_type::text,
        @name::text,
        @description::text,
        @avatar_url::text,
        @is_archived::bool,
        @is_pinned::bool,
        @is_muted::bool,
        @mute_until::timestamptz,
        @is_read_only::bool,
        @is_locked::bool,
        @unread_count::int,
        @unread_mention_count::int,
        @total_message_count::int,
        @last_message_at::timestamptz,
        @last_activity_at::timestamptz,
        @platform_metadata::jsonb
    ) ON CONFLICT (user_integration_id, external_conversation_id) DO
UPDATE
SET conversation_type = EXCLUDED.conversation_type,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    avatar_url = EXCLUDED.avatar_url,
    is_archived = EXCLUDED.is_archived,
    is_pinned = EXCLUDED.is_pinned,
    is_muted = EXCLUDED.is_muted,
    mute_until = EXCLUDED.mute_until,
    is_read_only = EXCLUDED.is_read_only,
    is_locked = EXCLUDED.is_locked,
    unread_count = EXCLUDED.unread_count,
    unread_mention_count = EXCLUDED.unread_mention_count,
    total_message_count = EXCLUDED.total_message_count,
    last_message_at = EXCLUDED.last_message_at,
    last_activity_at = EXCLUDED.last_activity_at,
    platform_metadata = EXCLUDED.platform_metadata,
    updated_at = NOW()
RETURNING id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at;
-- name: GetConversationByID :one
SELECT id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at
FROM conversations
WHERE id = $1::uuid;
-- name: ListUserIntegrationConversations :many
SELECT id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at
FROM conversations
WHERE user_integration_id = $1::int
ORDER BY last_activity_at DESC NULLS LAST;
-- name: ListActiveConversations :many
SELECT id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at
FROM conversations
WHERE user_integration_id = $1::int
    AND is_archived = false
ORDER BY last_activity_at DESC NULLS LAST
LIMIT $2::int OFFSET $3::int;
-- name: ListPinnedConversations :many
SELECT id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at
FROM conversations
WHERE user_integration_id = $1::int
    AND is_pinned = true
ORDER BY last_activity_at DESC NULLS LAST;
-- name: ListUnreadConversations :many
SELECT id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at
FROM conversations
WHERE user_integration_id = $1::int
    AND unread_count > 0
ORDER BY last_activity_at DESC;
-- name: UpdateConversationReadStatus :exec
UPDATE conversations
SET unread_count = $3::int,
    unread_mention_count = $4::int,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: IncrementConversationUnreadCount :exec
UPDATE conversations
SET unread_count = unread_count + 1,
    unread_mention_count = CASE
        WHEN $3::bool THEN unread_mention_count + 1
        ELSE unread_mention_count
    END,
    last_activity_at = $4::timestamptz,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: UpdateConversationMessageCount :exec
UPDATE conversations
SET total_message_count = $3::int,
    last_message_at = $4::timestamptz,
    last_activity_at = $4::timestamptz,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: DeleteConversation :exec
DELETE FROM conversations
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: ArchiveConversation :exec
UPDATE conversations
SET is_archived = true,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: UnarchiveConversation :exec
UPDATE conversations
SET is_archived = false,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: PinConversation :exec
UPDATE conversations
SET is_pinned = true,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: UnpinConversation :exec
UPDATE conversations
SET is_pinned = false,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: MuteConversation :exec
UPDATE conversations
SET is_muted = true,
    mute_until = $3::timestamptz,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: UnmuteConversation :exec
UPDATE conversations
SET is_muted = false,
    mute_until = NULL,
    updated_at = NOW()
WHERE user_integration_id = $1::int
    AND external_conversation_id = $2::text;
-- name: UpdateConversationState :exec
UPDATE conversations
SET is_archived = @is_archived::bool,
    is_pinned = @is_pinned::bool,
    is_muted = @is_muted::bool,
    mute_until = @mute_until::timestamptz,
    updated_at = NOW()
WHERE user_integration_id = @user_integration_id::int
    AND external_conversation_id = @external_conversation_id::text;
-- name: GetConversationByExternalID :one
SELECT id,
    user_integration_id,
    external_conversation_id,
    integration_type,
    conversation_type,
    name,
    description,
    avatar_url,
    is_archived,
    is_pinned,
    is_muted,
    mute_until,
    is_read_only,
    is_locked,
    unread_count,
    unread_mention_count,
    total_message_count,
    last_message_at,
    last_activity_at,
    platform_metadata,
    created_at,
    updated_at
FROM conversations
WHERE user_integration_id = @user_integration_id::int
    AND external_conversation_id = @external_conversation_id::text;