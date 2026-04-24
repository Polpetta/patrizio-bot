-- +goose Up
-- +goose StatementBegin
CREATE TABLE chat_settings (
    chat_id    INTEGER NOT NULL,
    key        TEXT    NOT NULL,
    value      TEXT    NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, key)
);
CREATE INDEX idx_chat_settings_chat_id ON chat_settings(chat_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_chat_settings_chat_id;
DROP TABLE IF EXISTS chat_settings;
-- +goose StatementEnd
