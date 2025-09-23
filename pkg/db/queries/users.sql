-- Users table queries
-- Authentication and user management

-- name: CreateUser :one
INSERT INTO users (
    username, email, password_hash, full_name
) VALUES (
    $1, $2, $3, $4
) RETURNING id, username, email, full_name, is_active, created_at, updated_at;

-- name: GetUserByUsername :one
SELECT id, username, email, password_hash, full_name, is_active, created_at, updated_at
FROM users 
WHERE username = $1 AND is_active = true;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, full_name, is_active, created_at, updated_at
FROM users 
WHERE email = $1 AND is_active = true;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, full_name, is_active, created_at, updated_at
FROM users 
WHERE id = $1 AND is_active = true;

-- name: GetUserByUsernameOrEmail :one
SELECT id, username, email, password_hash, full_name, is_active, created_at, updated_at
FROM users 
WHERE (username = $1 OR email = $1) AND is_active = true;

-- name: UpdateUser :one
UPDATE users 
SET 
    email = COALESCE($2, email),
    full_name = COALESCE($3, full_name),
    updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, username, email, full_name, is_active, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users 
SET password_hash = $2, updated_at = NOW()
WHERE id = $1 AND is_active = true;

-- name: DeactivateUser :exec
UPDATE users 
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, full_name, is_active, created_at, updated_at
FROM users 
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) as total_users
FROM users 
WHERE is_active = true;

-- name: CheckUsernameExists :one
SELECT EXISTS(
    SELECT 1 FROM users 
    WHERE username = $1 AND is_active = true
) as exists;

-- name: CheckEmailExists :one
SELECT EXISTS(
    SELECT 1 FROM users 
    WHERE email = $1 AND is_active = true
) as exists;
