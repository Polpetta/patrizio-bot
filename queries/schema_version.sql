-- name: GetSchemaVersion :one
-- Retrieves the current schema version from the database.
SELECT id, applied_at FROM schema_version ORDER BY id DESC LIMIT 1;
