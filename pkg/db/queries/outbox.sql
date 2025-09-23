-- Outbox table queries
-- Reliable message sending queue

-- name: CreateOutboxEntry :one
INSERT INTO outbox (
    client_msg_uuid, account_id, convo_id, server_msg_id, status
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (client_msg_uuid) DO NOTHING
RETURNING client_msg_uuid, created_at;

-- name: GetPendingOutboxEntries :many
SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
FROM outbox 
WHERE status IN ('queued', 'retry')
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdateOutboxStatus :exec
UPDATE outbox 
SET status = $2, last_error = $3, updated_at = NOW()
WHERE client_msg_uuid = $1;

-- name: GetOutboxEntry :one
SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
FROM outbox 
WHERE client_msg_uuid = $1;

-- name: GetOutboxByServerMsgID :one
SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
FROM outbox 
WHERE server_msg_id = $1;

-- name: DeleteOutboxEntry :exec
DELETE FROM outbox 
WHERE client_msg_uuid = $1;

-- name: GetFailedOutboxEntries :many
SELECT client_msg_uuid, account_id, convo_id, server_msg_id, status, last_error, created_at, updated_at
FROM outbox 
WHERE status = 'failed' AND created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- name: RetryOutboxEntry :exec
UPDATE outbox 
SET status = 'retry', last_error = NULL, updated_at = NOW()
WHERE client_msg_uuid = $1 AND status = 'failed';
