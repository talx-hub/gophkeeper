-- name: InsertBlob :one
INSERT INTO secret_blobs(objects_id, sealed)
VALUES ($1, $2)
RETURNING id;

-- name: PutManifest :exec
INSERT INTO chunk_manifest(objects_id, blob_id, chunk_index, length)
VALUES ($1, $2, $3, $4);

-- name: DeleteObject :exec
DELETE FROM secret_blobs
WHERE cm.objects_id = (
    SELECT id FROM objects
    WHERE storage_locator = $1
);
