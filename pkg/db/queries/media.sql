-- Media blobs table queries  
-- Content-addressed media storage

-- name: InsertMediaBlob :one
INSERT INTO media_blobs (
    content_hash, mime_type, size_bytes, storage_url
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (content_hash) DO NOTHING
RETURNING content_hash, created_at;

-- name: GetMediaBlob :one
SELECT content_hash, mime_type, size_bytes, storage_url, created_at
FROM media_blobs 
WHERE content_hash = $1;

-- name: DeleteMediaBlob :exec
DELETE FROM media_blobs 
WHERE content_hash = $1;

-- name: ListMediaBlobsBySize :many
SELECT content_hash, mime_type, size_bytes, storage_url, created_at
FROM media_blobs 
WHERE size_bytes > $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetTotalMediaStorage :one
SELECT COALESCE(SUM(size_bytes), 0) as total_bytes
FROM media_blobs;
