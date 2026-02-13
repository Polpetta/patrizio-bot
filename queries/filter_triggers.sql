-- name: InsertFilterTrigger :exec
INSERT INTO filter_triggers (filter_id, trigger_text)
VALUES (?1, ?2);

-- name: DeleteTriggerByChatAndText :one
DELETE FROM filter_triggers
WHERE id IN (
    SELECT ft.id
    FROM filter_triggers ft
    JOIN filters f ON ft.filter_id = f.id
    WHERE f.chat_id = ?1 AND ft.trigger_text = ?2
    LIMIT 1
)
RETURNING filter_id;

-- name: CountTriggersByFilterID :one
SELECT COUNT(*) FROM filter_triggers
WHERE filter_id = ?1;

-- name: CheckDuplicateTriggerInChat :one
SELECT EXISTS(
    SELECT 1
    FROM filter_triggers ft
    JOIN filters f ON ft.filter_id = f.id
    WHERE f.chat_id = ?1 AND ft.trigger_text = ?2
) AS has_duplicate;

-- name: GetTriggersByFilterID :many
SELECT id, filter_id, trigger_text
FROM filter_triggers
WHERE filter_id = ?1
ORDER BY id;
