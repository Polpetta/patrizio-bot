-- name: InsertConversationMessage :exec
INSERT INTO conversation_messages (thread_root_id, msg_id, parent_msg_id, role, content, sender_name)
VALUES (?, ?, ?, ?, ?, ?);

-- name: IsConversationMessage :one
SELECT thread_root_id FROM conversation_messages WHERE msg_id = ?;

-- name: GetThreadChain :many
WITH RECURSIVE chain AS (
    SELECT cm.id, cm.thread_root_id, cm.msg_id, cm.parent_msg_id, cm.role, cm.content, cm.sender_name, cm.created_at
    FROM conversation_messages cm
    WHERE cm.msg_id = @leaf_msg_id
    UNION ALL
    SELECT cm2.id, cm2.thread_root_id, cm2.msg_id, cm2.parent_msg_id, cm2.role, cm2.content, cm2.sender_name, cm2.created_at
    FROM conversation_messages cm2
    INNER JOIN chain c ON cm2.msg_id = c.parent_msg_id
)
SELECT sub.role, sub.content, sub.sender_name FROM (
    SELECT chain.id, chain.role, chain.content, chain.sender_name FROM chain
    ORDER BY chain.id DESC
    LIMIT @max_messages
) sub
ORDER BY sub.id ASC;
