-- name: InsertFilter :one
INSERT INTO filters (chat_id, response_type)
VALUES (?1, ?2)
RETURNING id;

-- name: DeleteFilter :exec
DELETE FROM filters WHERE id = ?1;

-- name: ListFiltersByChatID :many
SELECT id, chat_id, response_type, created_at
FROM filters
WHERE chat_id = ?1
ORDER BY created_at DESC;

-- name: GetFilterByID :one
SELECT id, chat_id, response_type, created_at
FROM filters
WHERE id = ?1;
