-- name: Create :one
INSERT INTO users(login_hash, password_phc)
VALUES ($1, $2)
RETURNING id;

-- name: FindByLogin :one
SELECT id, password_phc
FROM users
WHERE login_hash == $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
