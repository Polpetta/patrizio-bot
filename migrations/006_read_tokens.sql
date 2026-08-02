-- +goose Up
-- +goose StatementBegin
CREATE TABLE memory_read_tokens (
    msg_id         INTEGER PRIMARY KEY,
    observed_sha   TEXT    NOT NULL,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS memory_read_tokens;
-- +goose StatementEnd
