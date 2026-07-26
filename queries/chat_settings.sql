-- name: GetChatSetting :one
SELECT value FROM chat_settings WHERE chat_id = ? AND key = ?;

-- name: UpsertChatSetting :exec
INSERT INTO chat_settings (chat_id, key, value)
VALUES (?, ?, ?)
ON CONFLICT (chat_id, key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP;

-- name: DeleteChatSetting :exec
DELETE FROM chat_settings WHERE chat_id = ? AND key = ?;
