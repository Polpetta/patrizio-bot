-- name: GetReadToken :one
WITH RECURSIVE branch(msg_id, parent_msg_id, depth) AS (
    SELECT c.msg_id, c.parent_msg_id, 0
    FROM conversation_messages c
    WHERE c.msg_id = ?          -- current turn's msg_id
    UNION ALL
    SELECT cm.msg_id, cm.parent_msg_id, b.depth + 1
    FROM conversation_messages cm
    JOIN branch b ON cm.msg_id = b.parent_msg_id
)
SELECT t.observed_sha
FROM branch b
JOIN memory_read_tokens t ON t.msg_id = b.msg_id
ORDER BY b.depth ASC
LIMIT 1;

-- name: SaveToken :exec
INSERT INTO memory_read_tokens (msg_id, observed_sha)
VALUES (?, ?);
