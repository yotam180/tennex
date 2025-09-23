-- Events table queries
-- Core queries for the append-only event log

-- name: InsertEvent :one
INSERT INTO events (
    id, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) ON CONFLICT (id) DO NOTHING
RETURNING seq, ts;

-- name: GetEventsSince :many
SELECT seq, id, ts, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
FROM events 
WHERE account_id = $1 AND seq > $2
ORDER BY seq ASC
LIMIT $3;

-- name: GetEventsByConvo :many
SELECT seq, id, ts, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
FROM events 
WHERE convo_id = $1 AND seq > $2
ORDER BY seq ASC
LIMIT $3;

-- name: GetLatestEventSeq :one
SELECT COALESCE(MAX(seq), 0) as latest_seq
FROM events 
WHERE account_id = $1;

-- name: GetEventByID :one
SELECT seq, id, ts, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
FROM events 
WHERE id = $1;

-- name: GetEventByWAMessageID :one
SELECT seq, id, ts, type, account_id, device_id, convo_id, wa_message_id, sender_jid, payload, attachment_ref
FROM events 
WHERE wa_message_id = $1 AND account_id = $2;

-- name: CountEventsByAccount :one
SELECT COUNT(*) as total_events
FROM events 
WHERE account_id = $1;

-- name: CountEventsByType :one  
SELECT COUNT(*) as total_events
FROM events 
WHERE type = $1 AND account_id = $2;
