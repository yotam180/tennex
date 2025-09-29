-- Conversations table queries
-- Platform-agnostic conversation/chat/channel management

-- name: UpsertConversation :one
INSERT INTO conversations (
    user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata
) VALUES (
    $1::int, $2::text, $3::text, $4::text, $5::text, $6::text, $7::text,
    $8::bool, $9::bool, $10::bool, $11::timestamptz, $12::bool, $13::bool,
    $14::int, $15::int, $16::int, $17::timestamptz, $18::timestamptz, $19::jsonb
) ON CONFLICT (user_integration_id, external_conversation_id) DO UPDATE SET
    conversation_type = EXCLUDED.conversation_type,
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
RETURNING id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at;

-- name: GetConversationByExternalID :one
SELECT id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at
FROM conversations 
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: GetConversationByID :one
SELECT id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at
FROM conversations 
WHERE id = $1::uuid;

-- name: ListUserIntegrationConversations :many
SELECT id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at
FROM conversations 
WHERE user_integration_id = $1::int 
ORDER BY last_activity_at DESC NULLS LAST;

-- name: ListActiveConversations :many
SELECT id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at
FROM conversations 
WHERE user_integration_id = $1::int AND is_archived = false
ORDER BY last_activity_at DESC NULLS LAST
LIMIT $2::int OFFSET $3::int;

-- name: ListPinnedConversations :many
SELECT id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at
FROM conversations 
WHERE user_integration_id = $1::int AND is_pinned = true
ORDER BY last_activity_at DESC NULLS LAST;

-- name: ListUnreadConversations :many
SELECT id, user_integration_id, external_conversation_id, integration_type, conversation_type,
    name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
    is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
    last_message_at, last_activity_at, platform_metadata, created_at, updated_at
FROM conversations 
WHERE user_integration_id = $1::int AND unread_count > 0
ORDER BY last_activity_at DESC;

-- name: UpdateConversationState :exec
UPDATE conversations 
SET is_archived = $3::bool, is_pinned = $4::bool, is_muted = $5::bool, 
    mute_until = $6::timestamptz, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: UpdateConversationReadStatus :exec
UPDATE conversations 
SET unread_count = $3::int, unread_mention_count = $4::int, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: IncrementConversationUnreadCount :exec
UPDATE conversations 
SET unread_count = unread_count + 1, 
    unread_mention_count = CASE WHEN $3::bool THEN unread_mention_count + 1 ELSE unread_mention_count END,
    last_activity_at = $4::timestamptz,
    updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: UpdateConversationMessageCount :exec
UPDATE conversations 
SET total_message_count = $3::int, last_message_at = $4::timestamptz, 
    last_activity_at = $4::timestamptz, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: DeleteConversation :exec
DELETE FROM conversations 
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: ArchiveConversation :exec
UPDATE conversations 
SET is_archived = true, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: UnarchiveConversation :exec
UPDATE conversations 
SET is_archived = false, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: PinConversation :exec
UPDATE conversations 
SET is_pinned = true, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: UnpinConversation :exec
UPDATE conversations 
SET is_pinned = false, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: MuteConversation :exec
UPDATE conversations 
SET is_muted = true, mute_until = $3::timestamptz, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;

-- name: UnmuteConversation :exec
UPDATE conversations 
SET is_muted = false, mute_until = NULL, updated_at = NOW()
WHERE user_integration_id = $1::int AND external_conversation_id = $2::text;
