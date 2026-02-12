-- +goose Up
-- Initial migration to verify the migration system works.
-- This table will be replaced/extended by future migrations.

CREATE TABLE IF NOT EXISTS schema_version (
    id INTEGER PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_version (id) VALUES (1);
