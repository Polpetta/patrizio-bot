-- name: GetReadToken :one
-- Walk up the conversation ancestry from the current msg_id, returning the
-- nearest saved read-token. The seed row uses the parameter directly (rather
-- than joining conversation_messages) so this also works during the first
-- turn of a /prompt, when the current msg_id has not yet been persisted into
-- conversation_messages (that insert happens after ChatCompletion returns).
WITH RECURSIVE branch(msg_id, parent_msg_id, depth) AS (
    SELECT
        CAST(?1 AS INTEGER) AS msg_id,
        (SELECT c.parent_msg_id FROM conversation_messages c WHERE c.msg_id = ?1) AS parent_msg_id,
        0 AS depth
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
VALUES (?1, ?2)
ON CONFLICT (msg_id) DO UPDATE SET observed_sha = ?2;
