-- name: CreateToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1, NOW(), NOW(), $2, NOW() + INTERVAL '60 day'
)
RETURNING *;

-- name: RevokeToken :exec
UPDATE refresh_tokens 
SET updated_at = NOW(), revoked_at = NOW()
WHERE token = $1;

-- name: FindByParams :one
SELECT * FROM refresh_tokens
WHERE (token IS NULL OR token = $1)
    AND (revoked_at IS NULL);
