-- Messages table queries
-- Platform-agnostic message storage and retrieval

-- name: UpsertMessage :one
INSERT INTO messages (
    conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata
) VALUES (
    $1::uuid, $2::text, $3::text, $4::text, $5::text, $6::text, $7::text, $8::text,
    $9::timestamptz, $10::timestamptz, $11::bool, $12::bool, $13::bool, $14::timestamptz,
    $15::uuid, $16::text, $17::text, $18::jsonb
) ON CONFLICT (conversation_id, external_message_id) DO UPDATE SET
    external_server_id = EXCLUDED.external_server_id,
    sender_display_name = EXCLUDED.sender_display_name,
    message_type = EXCLUDED.message_type,
    content = EXCLUDED.content,
    edit_timestamp = EXCLUDED.edit_timestamp,
    is_forwarded = EXCLUDED.is_forwarded,
    is_deleted = EXCLUDED.is_deleted,
    deleted_at = EXCLUDED.deleted_at,
    reply_to_message_id = EXCLUDED.reply_to_message_id,
    reply_to_external_id = EXCLUDED.reply_to_external_id,
    delivery_status = EXCLUDED.delivery_status,
    platform_metadata = EXCLUDED.platform_metadata,
    updated_at = NOW()
RETURNING id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at;

-- name: GetMessageByExternalID :one
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND external_message_id = $2::text;

-- name: GetMessageByID :one
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE id = $1::uuid;

-- name: ListConversationMessages :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND is_deleted = false
ORDER BY timestamp DESC
LIMIT $2::int OFFSET $3::int;

-- name: ListConversationMessagesAfter :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND timestamp > $2::timestamptz AND is_deleted = false
ORDER BY timestamp ASC
LIMIT $3::int;

-- name: ListConversationMessagesBefore :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND timestamp < $2::timestamptz AND is_deleted = false
ORDER BY timestamp DESC
LIMIT $3::int;

-- name: ListUnreadMessages :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND delivery_status != 'read' AND is_from_me = false AND is_deleted = false
ORDER BY timestamp ASC;

-- name: GetLatestMessageInConversation :one
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND is_deleted = false
ORDER BY timestamp DESC
LIMIT 1;

-- name: SearchMessagesInConversation :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND content ILIKE '%' || $2::text || '%' AND is_deleted = false
ORDER BY timestamp DESC
LIMIT $3::int;

-- name: ListMessagesByType :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE conversation_id = $1::uuid AND message_type = $2::text AND is_deleted = false
ORDER BY timestamp DESC
LIMIT $3::int OFFSET $4::int;

-- name: ListReplies :many
SELECT id, conversation_id, external_message_id, external_server_id, integration_type,
    sender_external_id, sender_display_name, message_type, content, timestamp,
    edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
    reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
    created_at, updated_at
FROM messages 
WHERE reply_to_message_id = $1::uuid AND is_deleted = false
ORDER BY timestamp ASC;

-- name: UpdateMessageDeliveryStatus :exec
UPDATE messages 
SET delivery_status = $2::text, updated_at = NOW()
WHERE external_message_id = $1::text;

-- name: MarkMessageAsRead :exec
UPDATE messages 
SET delivery_status = 'read', updated_at = NOW()
WHERE conversation_id = $1::uuid AND external_message_id = $2::text;

-- name: MarkConversationMessagesAsRead :exec
UPDATE messages 
SET delivery_status = 'read', updated_at = NOW()
WHERE conversation_id = $1::uuid AND delivery_status != 'read' AND is_from_me = false;

-- name: EditMessage :exec
UPDATE messages 
SET content = $3::text, edit_timestamp = $4::timestamptz, updated_at = NOW()
WHERE conversation_id = $1::uuid AND external_message_id = $2::text;

-- name: DeleteMessage :exec
UPDATE messages 
SET is_deleted = true, deleted_at = NOW(), updated_at = NOW()
WHERE conversation_id = $1::uuid AND external_message_id = $2::text;

-- name: RestoreMessage :exec
UPDATE messages 
SET is_deleted = false, deleted_at = NULL, updated_at = NOW()
WHERE conversation_id = $1::uuid AND external_message_id = $2::text;

-- name: CountConversationMessages :one
SELECT COUNT(*) 
FROM messages 
WHERE conversation_id = $1::uuid AND is_deleted = false;

-- name: CountUnreadMessages :one
SELECT COUNT(*) 
FROM messages 
WHERE conversation_id = $1::uuid AND delivery_status != 'read' AND is_from_me = false AND is_deleted = false;

-- name: GetMessageStats :one
SELECT 
    COUNT(*) as total_messages,
    COUNT(*) FILTER (WHERE is_from_me = true) as sent_messages,
    COUNT(*) FILTER (WHERE is_from_me = false) as received_messages,
    COUNT(*) FILTER (WHERE delivery_status != 'read' AND is_from_me = false) as unread_messages,
    COUNT(*) FILTER (WHERE message_type = 'image') as image_messages,
    COUNT(*) FILTER (WHERE message_type = 'video') as video_messages,
    COUNT(*) FILTER (WHERE message_type = 'audio') as audio_messages
FROM messages 
WHERE conversation_id = $1::uuid AND is_deleted = false;
