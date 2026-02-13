-- +goose Up
-- +goose StatementBegin
CREATE TABLE filters (
    id            INTEGER PRIMARY KEY,
    chat_id       INTEGER NOT NULL,
    response_type TEXT    NOT NULL,  -- 'text', 'media', 'reaction'
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_filters_chat_id ON filters(chat_id);

CREATE TABLE filter_triggers (
    id           INTEGER PRIMARY KEY,
    filter_id    INTEGER NOT NULL REFERENCES filters(id) ON DELETE CASCADE,
    trigger_text TEXT    NOT NULL,  -- stored lowercased, Unicode letters/digits/spaces only
    UNIQUE(filter_id, trigger_text)
);

CREATE TABLE filter_text_resp (
    filter_id     INTEGER PRIMARY KEY REFERENCES filters(id) ON DELETE CASCADE,
    response_text TEXT NOT NULL
);

CREATE TABLE filter_media_resp (
    filter_id  INTEGER PRIMARY KEY REFERENCES filters(id) ON DELETE CASCADE,
    media_hash TEXT NOT NULL,  -- SHA-512 hex, references file on disk
    media_type TEXT NOT NULL   -- 'image', 'sticker', 'gif', 'video'
);

CREATE TABLE filter_reaction_resp (
    filter_id INTEGER PRIMARY KEY REFERENCES filters(id) ON DELETE CASCADE,
    reaction  TEXT NOT NULL    -- emoji
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS filter_reaction_resp;
DROP TABLE IF EXISTS filter_media_resp;
DROP TABLE IF EXISTS filter_text_resp;
DROP TABLE IF EXISTS filter_triggers;
DROP TABLE IF EXISTS filters;
-- +goose StatementEnd
