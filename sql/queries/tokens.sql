-- name: Save :exec
INSERT INTO refresh_tokens(user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (token_hash) DO UPDATE
    SET user_id = EXCLUDED.user_id,
        expires_at = EXCLUDED.expires_at;

-- name: Validate :one
SELECT EXISTS (
    SELECT 1
    FROM refresh_tokens
    WHERE token_hash = $1 AND
          user_id = $2 AND
          expires_at > NOW()
) AS found;

-- name: DeleteToken :exec
DELETE FROM refresh_tokens
WHERE token_hash = $1;
