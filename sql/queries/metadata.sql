-- name: PutMetadata :one
INSERT INTO objects (user_id, id_data_type, name, description, storage_locator)
VALUES (
           sqlc.arg(user_id)::uuid,
           (SELECT id
            FROM data_types
            WHERE public.data_types.name = sqlc.arg(data_type_name)),
           sqlc.arg(object_name),
           sqlc.arg(description),
           sqlc.arg(storage_locator)
       )
RETURNING id;

-- name: GetMetadata :one
SELECT
    o.id,
    o.storage_locator,
    dt.name AS data_type_name,
    o.name AS object_name,
    o.description,
    o.created_at
FROM objects o
    JOIN public.data_types dt on o.id_data_type = dt.id
WHERE o.id = $1;

-- name: ListByUser :many
SELECT
    o.id,
    o.storage_locator,
    dt.name AS data_type_name,
    o.name AS object_name,
    o.description,
    o.created_at
FROM objects o
    JOIN public.data_types dt on o.id_data_type = dt.id
WHERE o.user_id = $1;

-- name: DeleteMetadata :one
DELETE FROM objects
WHERE id = $1
RETURNING storage_locator;
