-- +goose Up
-- +goose StatementBegin
CREATE TABLE conversation_messages (
    id             INTEGER PRIMARY KEY,
    thread_root_id INTEGER NOT NULL,   -- MsgId of the original /prompt message
    msg_id         INTEGER NOT NULL UNIQUE, -- Delta Chat MsgId
    parent_msg_id  INTEGER,            -- MsgId of the quoted/parent message, NULL for root
    role           TEXT    NOT NULL,    -- 'user' or 'assistant'
    content        TEXT    NOT NULL,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_conversation_messages_thread_root_id ON conversation_messages(thread_root_id);
CREATE INDEX idx_conversation_messages_msg_id ON conversation_messages(msg_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS conversation_messages;
-- +goose StatementEnd
