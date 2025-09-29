-- Message media table queries
-- Rich media attachment management

-- name: CreateMessageMedia :one
INSERT INTO message_media (
    message_id, media_type, file_name, file_size, mime_type, duration_seconds,
    width, height, original_url, thumbnail_url, local_file_path,
    download_status, downloaded_at, platform_metadata
) VALUES (
    $1::uuid, $2::text, $3::text, $4::bigint, $5::text, $6::int,
    $7::int, $8::int, $9::text, $10::text, $11::text,
    $12::text, $13::timestamptz, $14::jsonb
) RETURNING id, message_id, media_type, file_name, file_size, mime_type,
    duration_seconds, width, height, original_url, thumbnail_url,
    local_file_path, download_status, downloaded_at, platform_metadata,
    created_at, updated_at;

-- name: GetMessageMediaByID :one
SELECT id, message_id, media_type, file_name, file_size, mime_type,
    duration_seconds, width, height, original_url, thumbnail_url,
    local_file_path, download_status, downloaded_at, platform_metadata,
    created_at, updated_at
FROM message_media 
WHERE id = $1::uuid;

-- name: GetMessageMedia :one
SELECT id, message_id, media_type, file_name, file_size, mime_type,
    duration_seconds, width, height, original_url, thumbnail_url,
    local_file_path, download_status, downloaded_at, platform_metadata,
    created_at, updated_at
FROM message_media 
WHERE message_id = $1::uuid;

-- name: ListMessageMedia :many
SELECT id, message_id, media_type, file_name, file_size, mime_type,
    duration_seconds, width, height, original_url, thumbnail_url,
    local_file_path, download_status, downloaded_at, platform_metadata,
    created_at, updated_at
FROM message_media 
WHERE message_id = $1::uuid
ORDER BY created_at ASC;

-- name: ListConversationMedia :many
SELECT mm.id, mm.message_id, mm.media_type, mm.file_name, mm.file_size, mm.mime_type,
    mm.duration_seconds, mm.width, mm.height, mm.original_url, mm.thumbnail_url,
    mm.local_file_path, mm.download_status, mm.downloaded_at, mm.platform_metadata,
    mm.created_at, mm.updated_at
FROM message_media mm
INNER JOIN messages m ON mm.message_id = m.id
WHERE m.conversation_id = $1::uuid
ORDER BY m.timestamp DESC
LIMIT $2::int OFFSET $3::int;

-- name: ListConversationMediaByType :many
SELECT mm.id, mm.message_id, mm.media_type, mm.file_name, mm.file_size, mm.mime_type,
    mm.duration_seconds, mm.width, mm.height, mm.original_url, mm.thumbnail_url,
    mm.local_file_path, mm.download_status, mm.downloaded_at, mm.platform_metadata,
    mm.created_at, mm.updated_at
FROM message_media mm
INNER JOIN messages m ON mm.message_id = m.id
WHERE m.conversation_id = $1::uuid AND mm.media_type = $2::text
ORDER BY m.timestamp DESC
LIMIT $3::int OFFSET $4::int;

-- name: ListPendingDownloads :many
SELECT id, message_id, media_type, file_name, file_size, mime_type,
    duration_seconds, width, height, original_url, thumbnail_url,
    local_file_path, download_status, downloaded_at, platform_metadata,
    created_at, updated_at
FROM message_media 
WHERE download_status = 'pending'
ORDER BY created_at ASC
LIMIT $1::int;

-- name: ListFailedDownloads :many
SELECT id, message_id, media_type, file_name, file_size, mime_type,
    duration_seconds, width, height, original_url, thumbnail_url,
    local_file_path, download_status, downloaded_at, platform_metadata,
    created_at, updated_at
FROM message_media 
WHERE download_status = 'failed'
ORDER BY created_at DESC
LIMIT $1::int;

-- name: UpdateMediaInfo :exec
UPDATE message_media 
SET file_name = $2::text, file_size = $3::bigint, mime_type = $4::text,
    duration_seconds = $5::int, width = $6::int, height = $7::int,
    platform_metadata = $8::jsonb, updated_at = NOW()
WHERE id = $1::uuid;

-- name: UpdateDownloadStatus :exec
UPDATE message_media 
SET download_status = $2::text, updated_at = NOW()
WHERE id = $1::uuid;

-- name: MarkAsDownloaded :exec
UPDATE message_media 
SET download_status = 'completed', local_file_path = $2::text,
    downloaded_at = NOW(), updated_at = NOW()
WHERE id = $1::uuid;

-- name: MarkDownloadFailed :exec
UPDATE message_media 
SET download_status = 'failed', updated_at = NOW()
WHERE id = $1::uuid;

-- name: StartDownload :exec
UPDATE message_media 
SET download_status = 'downloading', updated_at = NOW()
WHERE id = $1::uuid;

-- name: UpdateLocalFilePath :exec
UPDATE message_media 
SET local_file_path = $2::text, updated_at = NOW()
WHERE id = $1::uuid;

-- name: DeleteMessageMedia :exec
DELETE FROM message_media 
WHERE id = $1::uuid;

-- name: DeleteMessageMediaByMessage :exec
DELETE FROM message_media 
WHERE message_id = $1::uuid;

-- name: CountMediaByType :one
SELECT 
    COUNT(*) FILTER (WHERE media_type = 'image') as images,
    COUNT(*) FILTER (WHERE media_type = 'video') as videos,
    COUNT(*) FILTER (WHERE media_type = 'audio') as audio,
    COUNT(*) FILTER (WHERE media_type = 'document') as documents,
    COUNT(*) FILTER (WHERE media_type = 'sticker') as stickers
FROM message_media mm
INNER JOIN messages m ON mm.message_id = m.id
WHERE m.conversation_id = $1::uuid;

-- name: GetMediaStorageStats :one
SELECT 
    COUNT(*) as total_files,
    COALESCE(SUM(file_size), 0) as total_bytes,
    COUNT(*) FILTER (WHERE download_status = 'completed') as downloaded_files,
    COUNT(*) FILTER (WHERE download_status = 'pending') as pending_downloads,
    COUNT(*) FILTER (WHERE download_status = 'failed') as failed_downloads
FROM message_media mm
INNER JOIN messages m ON mm.message_id = m.id
WHERE m.conversation_id = $1::uuid;
