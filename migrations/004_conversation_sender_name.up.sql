-- +goose Up
-- +goose StatementBegin
ALTER TABLE conversation_messages ADD COLUMN sender_name TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE conversation_messages DROP COLUMN sender_name;
-- +goose StatementEnd
