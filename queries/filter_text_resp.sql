-- name: InsertTextResponse :exec
INSERT INTO filter_text_resp (filter_id, response_text)
VALUES (?1, ?2);

-- name: GetTextResponseByFilterID :one
SELECT filter_id, response_text
FROM filter_text_resp
WHERE filter_id = ?1;
