## 1. Domain Layer: Ports and Models

- [x] 1.1 Create `internal/domain/models.go` with domain types: `Filter`, `FilterResponse`, `FilterTrigger`, response type constants (`text`, `media`, `reaction`), media type constants (`image`, `sticker`, `gif`, `video`)
- [x] 1.2 Create `internal/domain/ports.go` with port interfaces: `FilterRepository`, `MediaStorage`, `Config`
- [x] 1.3 Create `internal/domain/filter.go` with pure domain functions: `NormalizeMessage(msg string) string` (regex replace non-`\p{L}\p{N}` with space + lowercase), `ValidateTrigger(text string) error` (regex `^[\p{L}\p{N}\s]+$`), `NormalizeTrigger(text string) string` (lowercase)
- [x] 1.4 Create `internal/domain/deps.go` with `Dependencies` struct holding `FilterRepository`, `MediaStorage`, `Config`
- [x] 1.5 Write unit tests for `NormalizeMessage`, `ValidateTrigger`, `NormalizeTrigger` in `internal/domain/filter_test.go` covering Latin, CJK, Cyrillic, punctuation stripping, multi-space handling, and rejection of invalid triggers like `c++`

## 2. Database: Schema Migration and sqlc Queries

- [x] 2.1 Create goose migration `migrations/002_filters.up.sql` with the five tables (`filters`, `filter_triggers`, `filter_text_resp`, `filter_media_resp`, `filter_reaction_resp`), indexes, and constraints per the design schema
- [x] 2.2 Add `PRAGMA foreign_keys = ON` to `internal/database/database.go` `Open()` function after the WAL mode pragma
- [x] 2.3 Write sqlc query files in `queries/`: `filters.sql` (insert filter, delete filter by id, list filters by chat_id), `filter_triggers.sql` (insert trigger, delete trigger by chat+text, count triggers by filter_id, check duplicate trigger in chat), `filter_text_resp.sql` (insert), `filter_media_resp.sql` (insert, count by media_hash), `filter_reaction_resp.sql` (insert), `filter_matching.sql` (the CTE + UNION ALL matching query)
- [x] 2.4 Run `make sqlc` to generate Go code from the new query files and verify generation succeeds
- [x] 2.5 Update `sqlc.yaml` if needed (verify `queries/` glob and `migrations/` schema source cover the new files)

## 3. Config Refactor

- [x] 3.1 Define `Config` struct in `internal/config/config.go` that implements the domain `Config` port interface, with fields `DBPath`, `LogLevel`, `MediaPath`
- [x] 3.2 Add TOML config file loading to the `Load()` function: set config name/paths, add `media_path` default (`./media`), keep `PATRIZIO_` env var prefix with priority over TOML
- [x] 3.3 Return a `Config` struct from `Load()` instead of relying on global Viper state; update callers in `cmd/patrizio/main.go` and `internal/bot/bot.go`
- [x] 3.4 Write tests for config loading: defaults, TOML override, env var override priority

## 4. Media Storage Adapter

- [x] 4.1 Add `github.com/spf13/afero` as a direct dependency in `go.mod` (currently indirect)
- [x] 4.2 Create `internal/adapter/storage/storage.go` implementing the `MediaStorage` port interface using `afero.Fs`, with SHA-512 content-addressed filenames in the configured `media_path` directory
- [x] 4.3 Write tests for `MediaStorage` adapter using `afero.MemMapFs`: save, read, delete, exists, idempotent overwrite of same hash

## 5. SQLite FilterRepository Adapter

- [x] 5.1 Create `internal/adapter/sqlite/repository.go` implementing the `FilterRepository` port interface using sqlc-generated queries
- [x] 5.2 Implement `CreateTextFilter`: BEGIN tx, insert into `filters` + `filter_triggers` + `filter_text_resp`, check for duplicate triggers in chat, COMMIT (or rollback on error)
- [x] 5.3 Implement `CreateMediaFilter`: same as text but inserts into `filter_media_resp`
- [x] 5.4 Implement `CreateReactionFilter`: same as text but inserts into `filter_reaction_resp`
- [x] 5.5 Implement `RemoveTrigger`: delete trigger row, check if filter has remaining triggers, if not delete filter and return `media_hash` (if media type). Accept `MediaStorage` for transactional file cleanup
- [x] 5.6 Implement `RemoveAllFilters`: delete all filters for chat, return list of `media_hash` values from deleted media filters. Accept `MediaStorage` for transactional file cleanup
- [x] 5.7 Implement `ListFilters`: query all filters for a chat with their triggers and response types
- [x] 5.8 Implement `FindMatchingFilters`: execute the CTE + UNION ALL matching query with normalized message
- [x] 5.9 Write integration tests using in-memory SQLite (`:memory:`) with real schema migrations: test all CRUD operations, matching, cascading deletes, duplicate trigger rejection, transactional media cleanup

## 6. Command Parser

- [x] 6.1 Create `internal/domain/command.go` with command parsing functions: parse `/filter`, `/stop`, `/stopall`, `/filters` from message text. Handle single-word triggers, quoted multi-word triggers, parenthesized multi-trigger syntax, and `react:<emoji>` response type
- [x] 6.2 Write thorough unit tests for command parsing: single trigger, quoted phrase, multiple triggers with parentheses, mixed quoted/unquoted in parentheses, `react:` prefix, edge cases (empty input, missing args, unclosed quotes)

## 7. Bot Handler Wiring

- [x] 7.1 Refactor `internal/bot/bot.go` `Setup()` to accept or build a `Dependencies` struct; wire `*sql.DB` into the handler instead of discarding it
- [x] 7.2 Refactor `internal/bot/handler.go` `handleGroupMessage()`: normalize incoming message text, call `FindMatchingFilters`, iterate results and dispatch responses (text reply, media reply, reaction) via Delta Chat RPC
- [x] 7.3 Add command routing in `handleGroupMessage()`: detect `/filter`, `/stop`, `/stopall`, `/filters` prefixes and dispatch to the corresponding command handler functions
- [x] 7.4 Implement `/filter` command handler: parse command, validate triggers, determine response type (text, media from replied message, reaction), call appropriate `CreateXxxFilter`, send confirmation
- [x] 7.5 Implement `/stop` command handler: parse trigger, call `RemoveTrigger`, clean up media if hash returned, send confirmation or "not found"
- [x] 7.6 Implement `/stopall` command handler: call `RemoveAllFilters`, clean up media files for returned hashes, send confirmation
- [x] 7.7 Implement `/filters` command handler: call `ListFilters`, format and send list or "no filters" message
- [x] 7.8 Implement media download in `/filter` handler: when user replies to a media message, download the attachment via Delta Chat RPC, compute SHA-512 hash, save via `MediaStorage`, pass hash to `CreateMediaFilter`

## 8. Entrypoint and Dockerfile

- [x] 8.1 Update `cmd/patrizio/main.go` to build `Dependencies` from real adapters: SQLite `FilterRepository`, Afero `OsFs` `MediaStorage`, Viper+TOML `Config`
- [x] 8.2 Create media directory on startup if it doesn't exist (using the configured `media_path`)
- [x] 8.3 Add `VOLUME` directives to `Dockerfile` for database and media persistence

## 9. End-to-End Verification

- [x] 9.1 Run `make sqlc` and verify all queries generate without errors
- [x] 9.2 Run `make build` and verify the binary compiles
- [x] 9.3 Run `make test` and verify all unit and integration tests pass
- [x] 9.4 Run `make lint` and fix any linter warnings
