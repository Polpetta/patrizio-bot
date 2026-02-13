## Why

Patrizio currently has no way to respond to incoming group messages — `handleGroupMessage()` is a no-op placeholder. The bot needs a filter system that allows group members to define trigger words/phrases that produce automatic responses (text, media, or emoji reactions). This is the core feature that makes the bot useful, modeled after the Rose Telegram bot's filter system.

## What Changes

- Introduce a filter engine that matches incoming group messages against stored triggers and produces responses (text, media, or reactions)
- Add a table-per-type database schema: `filters`, `filter_triggers`, `filter_text_resp`, `filter_media_resp`, `filter_reaction_resp`
- Add media storage on disk using SHA-512 content hashing for deduplication, backed by Afero VFS for testability
- Introduce TOML config file support (with env var override) for settings like media storage path
- Introduce a `Dependencies` struct for dependency injection, following hexagonal architecture: domain ports (`FilterRepository`, `MediaStorage`, `Config`) with concrete adapters (SQLite, Afero, Viper+TOML)
- Add bot commands for filter management: `/filter`, `/stop`, `/stopall`, `/filters`
- Wire the database connection into the message handler (currently discarded with `_ = db`)
- Implement message normalization (strip non-Unicode-alphanumeric characters, lowercase) to support punctuation-insensitive matching
- All filter matching logic pushed to SQL (CTE + UNION ALL query), Go only handles input normalization
- Media file deletion happens inside SQLite transactions to prevent race conditions with concurrent inserts
- Refactor the config package from global Viper state to an injectable struct/interface

## Capabilities

### New Capabilities
- `filter-engine`: Core filter matching — trigger validation (Unicode `\p{L}`, `\p{N}`, spaces only), message normalization, SQL-based matching via `INSTR` with space-padding, response resolution across text/media/reaction types
- `filter-management`: Bot commands for filter CRUD — `/filter` (create with single-word, multi-word, or multiple triggers), `/stop` (remove a trigger), `/stopall` (remove all filters in a chat), `/filters` (list all filters in a chat)
- `media-storage`: SHA-512 content-addressed media storage with deduplication, backed by Afero VFS. Transactional cleanup on filter deletion to prevent orphaned files or race conditions
- `dependency-injection`: `Dependencies` struct holding port implementations (`FilterRepository`, `MediaStorage`, `Config`), wired at the entrypoint. Hexagonal architecture with domain ports and adapter implementations

### Modified Capabilities
- `database`: New migration adding five tables (`filters`, `filter_triggers`, `filter_text_resp`, `filter_media_resp`, `filter_reaction_resp`). Wire `*sql.DB` into the message handler
- `message-handling`: Route group messages through the filter engine instead of the current no-op. Bot quote-replies to triggering messages with the matched filter's response
- `bot-skeleton`: Refactor to accept `Dependencies` struct. Introduce TOML config file support alongside existing env vars (env vars take priority)

## Impact

- **New dependency**: `github.com/spf13/afero` for virtual filesystem
- **Config package refactor**: move from global Viper functions to an injectable `Config` struct. TOML file support added, env vars retain priority
- **Database schema**: five new tables, new migration file, new sqlc queries
- **Bot initialization**: `main.go` builds `Dependencies` and passes it through. `bot.Setup()` signature changes to accept dependencies
- **Message handler**: `handleGroupMessage()` goes from no-op to calling the filter domain logic
- **Filesystem**: new media directory managed by the bot, path configurable via `media_path` setting
- **Testing**: domain logic fully testable with in-memory SQLite + `MemMapFs` + static config, no external dependencies needed
