-- name: InsertMediaResponse :exec
INSERT INTO filter_media_resp (filter_id, media_hash, media_type)
VALUES (?1, ?2, ?3);

-- name: CountMediaResponsesByHash :one
SELECT COUNT(*) FROM filter_media_resp
WHERE media_hash = ?1;

-- name: GetMediaResponseByFilterID :one
SELECT filter_id, media_hash, media_type
FROM filter_media_resp
WHERE filter_id = ?1;

-- name: GetMediaHashesByChatID :many
SELECT DISTINCT media_hash
FROM filter_media_resp
WHERE filter_id IN (
    SELECT id FROM filters WHERE chat_id = ?1
);
