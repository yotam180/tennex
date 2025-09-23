-- Accounts table queries
-- WhatsApp account management

-- name: UpsertAccount :one
INSERT INTO accounts (
    id, wa_jid, display_name, avatar_url, status, last_seen
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (id) DO UPDATE SET
    wa_jid = EXCLUDED.wa_jid,
    display_name = EXCLUDED.display_name,
    avatar_url = EXCLUDED.avatar_url,
    status = EXCLUDED.status,
    last_seen = EXCLUDED.last_seen,
    updated_at = NOW()
RETURNING id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at;

-- name: GetAccount :one
SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
FROM accounts 
WHERE id = $1;

-- name: GetAccountByWAJID :one
SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
FROM accounts 
WHERE wa_jid = $1;

-- name: UpdateAccountStatus :exec
UPDATE accounts 
SET status = $2, last_seen = $3, updated_at = NOW()
WHERE id = $1;

-- name: ListAccounts :many
SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
FROM accounts 
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetConnectedAccounts :many
SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
FROM accounts 
WHERE status = 'connected'
ORDER BY last_seen DESC;

-- name: DeleteAccount :exec
DELETE FROM accounts 
WHERE id = $1;
