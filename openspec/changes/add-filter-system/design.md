## Context

Patrizio is a Delta Chat group bot with a skeleton already in place: message routing (`internal/bot/handler.go`), SQLite database with goose migrations (`internal/database/`), sqlc query generation, and Viper-based config from env vars (`internal/config/`). The group message handler (`handleGroupMessage`) is currently a no-op placeholder. The database connection opened in `OnBotStart` is discarded (`_ = db`) and not wired into the handler.

This design introduces the filter system — the bot's core feature — while restructuring the codebase to follow hexagonal architecture, making the domain logic fully testable without external dependencies.

## Goals / Non-Goals

**Goals:**
- Implement a filter engine that matches incoming group messages against stored triggers and responds with text, media, or emoji reactions
- Provide bot commands (`/filter`, `/stop`, `/stopall`, `/filters`) for any group member to manage filters
- Store media attachments on disk using SHA-512 content hashing for deduplication
- Introduce hexagonal architecture with dependency injection, keeping domain logic free of Delta Chat, SQLite, and filesystem imports
- Make the domain layer fully testable with in-memory SQLite, Afero `MemMapFs`, and static config
- Add TOML config file support alongside existing env vars

**Non-Goals:**
- Prefix or exact filter match modes (future work)
- Template variables / fillings (`{replytag}`, `{user}`, `{admin}`)
- Permission restrictions on who can trigger or manage filters
- In-memory caching of triggers per chat (SQLite with WAL is sufficient)
- Filter import/export functionality

## Decisions

### 1. Hexagonal architecture with a Dependencies struct

**Decision:** Introduce a `Dependencies` struct that holds port implementations (`FilterRepository`, `MediaStorage`, `Config`), injected at the entrypoint. The domain package defines interfaces (ports); adapters implement them.

**Rationale:** The current codebase uses global state for config (`viper.GetString(...)` in package-level functions) and discards the DB handle. This makes testing impossible without external resources. By defining ports as interfaces in the domain and injecting adapter implementations, we can test the full domain logic with in-memory fakes.

**Alternatives considered:**
- *Global singletons:* Current approach. Untestable, tight coupling.
- *Context-based injection:* Passing dependencies via `context.Value`. Type-unsafe, hard to trace.

**Structure:**

```
cmd/patrizio/main.go          — builds real Dependencies, wires everything
internal/
  domain/
    ports.go                   — FilterRepository, MediaStorage, Config interfaces
    filter.go                  — filter matching, trigger validation, normalization
    models.go                  — Filter, FilterResponse, etc.
  adapter/
    sqlite/                    — FilterRepository implementation using sqlc
    storage/                   — MediaStorage implementation using Afero
    config/                    — Config implementation using Viper + TOML
  bot/
    bot.go                     — inbound adapter, receives messages, calls domain
    handler.go                 — translates domain responses to Delta Chat RPC calls
```

The bot package is the inbound adapter: it receives Delta Chat messages, calls domain functions, and translates the returned response structs into RPC calls (`MiscSendTextMessage`, `SendReaction`, etc.). The domain package has zero imports from Delta Chat, Afero, or SQLite.

### 2. Table-per-type database schema

**Decision:** Use five tables: `filters` (with `response_type` discriminator), `filter_triggers`, `filter_text_resp`, `filter_media_resp`, `filter_reaction_resp`. Each response table uses `filter_id` as both PK and FK (1:0..1 relationship).

**Rationale:** Strong type enforcement at the schema level. A text filter cannot accidentally have a `media_hash` because that column does not exist in its table. No nullable columns that are "sometimes used" — each table only has NOT NULL columns relevant to its type. The `response_type` discriminator on `filters` enables efficient querying without probing all three response tables.

**Alternatives considered:**
- *Single table with nullable columns:* Three mutually exclusive nullable columns (`response_text`, `media_hash`, `reaction`). Simpler schema, but mutual exclusivity is only enforced in application code. Essentially NoSQL thinking in a relational DB.
- *Single polymorphic column:* One `response_value TEXT` column where meaning depends on `response_type`. Loses all type information.
- *Table-per-type without discriminator:* Same tables but no `response_type` on `filters`. Requires probing all three response tables or re-inventing the discriminator in UNION queries.

**Schema:**

```sql
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
```

### 3. Message normalization in Go, matching in SQL

**Decision:** Incoming messages are normalized in Go by lowercasing and replacing all non-Unicode-alphanumeric characters (`\p{L}`, `\p{N}`) with spaces. The normalized message is passed to a SQL query that uses `INSTR` with space-padding for whole-word matching.

**Rationale:** Pushing all matching logic into SQL keeps the domain layer thin and avoids duplicating matching rules between Go and SQL. Go only performs mechanical input sanitization (a single `regexp.ReplaceAllString` call) with no business logic. The space-padding trick (`INSTR(' ' || msg || ' ', ' ' || trigger || ' ')`) provides word-boundary matching that handles triggers at the start, end, or middle of a message.

This approach is safe because triggers are constrained to only contain Unicode alphanumeric characters and spaces (validated at creation time). Since the message is normalized to the same character space, the space-padding match works reliably with no edge cases around punctuation.

**Matching query (CTE + UNION ALL):**

```sql
WITH matched AS (
    SELECT f.id, f.response_type
    FROM filter_triggers t
    JOIN filters f ON t.filter_id = f.id
    WHERE f.chat_id = ?1
    AND INSTR(' ' || ?2 || ' ', ' ' || t.trigger_text || ' ') > 0
)
SELECT m.id, m.response_type, ft.response_text,
       NULL AS media_hash, NULL AS media_type, NULL AS reaction
FROM matched m
JOIN filter_text_resp ft ON m.id = ft.filter_id
WHERE m.response_type = 'text'

UNION ALL

SELECT m.id, m.response_type, NULL, fm.media_hash, fm.media_type, NULL
FROM matched m
JOIN filter_media_resp fm ON m.id = fm.filter_id
WHERE m.response_type = 'media'

UNION ALL

SELECT m.id, m.response_type, NULL, NULL, NULL, fr.reaction
FROM matched m
JOIN filter_reaction_resp fr ON m.id = fr.filter_id
WHERE m.response_type = 'reaction';
```

**Alternatives considered:**
- *Word-boundary matching in Go:* Load all triggers for a chat, iterate in Go code. Pushes business logic into Go unnecessarily.
- *SQLite FTS5:* Designed for indexing stored documents, not matching transient messages against stored triggers. Would require inverting the typical FTS pattern.
- *No normalization (pure substring):* `INSTR` on raw message text. "dog" would match "hotdog" — rejected.

### 4. Trigger validation: Unicode alphanumeric only

**Decision:** Filter triggers can only contain Unicode letters (`\p{L}`), Unicode digits (`\p{N}`), and spaces. All other characters (punctuation, symbols) are rejected at creation time. Triggers are stored lowercased.

**Rationale:** This constraint eliminates the collision problem where normalizing both message and trigger loses information. For example, `c++` would normalize to `c` and collide with a `c` trigger. By rejecting special characters in triggers, we ensure the normalized message and stored triggers always live in the same character space, making `INSTR` matching unambiguous.

This supports all natural language scripts (Latin, Japanese, Arabic, Cyrillic, Georgian, etc.) via Unicode character categories.

**Validation regex:** `^[\p{L}\p{N}\s]+$`

### 5. SHA-512 content-addressed media storage with Afero VFS

**Decision:** Media files are stored on disk with their SHA-512 hex digest as the filename, in a configurable directory. The filesystem is abstracted behind Afero's `afero.Fs` interface — `OsFs` in production, `MemMapFs` in tests.

**Rationale:** Content-addressed storage gives natural deduplication. If two filters reference the same image, only one copy exists on disk. Afero is already in the spf13 ecosystem (same author as Viper/Cobra) and provides a clean interface swap for testing without touching the real filesystem.

**FilterRepository port interface:**

```go
type FilterRepository interface {
    CreateTextFilter(ctx context.Context, chatID int64, triggers []string, responseText string) error
    CreateMediaFilter(ctx context.Context, chatID int64, triggers []string, mediaHash string, mediaType string) error
    CreateReactionFilter(ctx context.Context, chatID int64, triggers []string, reaction string) error
    RemoveTrigger(ctx context.Context, chatID int64, triggerText string) (*string, error)
    RemoveAllFilters(ctx context.Context, chatID int64) ([]string, error)
    ListFilters(ctx context.Context, chatID int64) ([]Filter, error)
    FindMatchingFilters(ctx context.Context, chatID int64, normalizedMessage string) ([]FilterResponse, error)
}
```

`RemoveTrigger` returns the `media_hash` (if any) of the deleted filter when it was the last trigger. `RemoveAllFilters` returns all `media_hash` values from deleted media filters. The caller uses these to coordinate media cleanup.

**MediaStorage port interface:**

```go
type MediaStorage interface {
    Save(hash string, data []byte) error
    Delete(hash string) error
    Read(hash string) ([]byte, error)
    Exists(hash string) (bool, error)
}
```

### 6. Transactional media cleanup on deletion

**Decision:** When deleting a filter with a media response, the file deletion happens inside the SQLite write transaction, before commit.

**Sequence:**

1. `BEGIN` — acquires SQLite write lock
2. `DELETE FROM filters WHERE id = ?` — cascade removes trigger and response rows
3. `SELECT COUNT(*) FROM filter_media_resp WHERE media_hash = ?` — check if hash still referenced
4. If count is 0, delete the file from disk via `MediaStorage.Delete(hash)`
5. `COMMIT` — releases the lock

**Rationale:** SQLite serializes all writers. By deleting the file before committing, no other transaction can insert a new reference to the same hash between the count check and the file deletion. This prevents:
- Orphaned files (delete race: two concurrent deletes both skip file deletion because each sees the other's reference)
- Missing files (delete-then-insert race: file deleted just before a new filter referencing it is committed)

The tradeoff is filesystem I/O inside a database transaction, making it slightly longer. For a Delta Chat bot's concurrency level this is negligible.

**On insertion**, media writes are idempotent: writing the same SHA-512 content to the same path simply overwrites identical bytes. No special handling needed for concurrent inserts with the same media.

### 7. TOML config file with env var override

**Decision:** Refactor the config package from package-level Viper functions to a `Config` struct that implements a domain port interface. Load from a TOML config file with env vars (`PATRIZIO_` prefix) taking priority.

**Rationale:** The current config uses global Viper state (`viper.GetString(...)` via package functions), which is untestable and incompatible with the DI approach. A `Config` struct can be constructed directly in tests with any values needed. TOML provides a human-readable config file format for settings like `media_path` that are awkward to manage solely through env vars.

**Config fields:**

| Field | TOML key | Env var | Default |
|---|---|---|---|
| `DBPath` | `db_path` | `PATRIZIO_DB_PATH` | `./patrizio.db` |
| `LogLevel` | `log_level` | `PATRIZIO_LOG_LEVEL` | `info` |
| `MediaPath` | `media_path` | `PATRIZIO_MEDIA_PATH` | `./media` |

**Priority:** env var > TOML file > default.

## Risks / Trade-offs

**[INSTR performance on large trigger sets]** The matching query scans all triggers for a chat on every incoming message using `INSTR`, which is O(N) in the number of triggers per chat. For typical group chats (tens to low hundreds of filters), this is fine with SQLite's WAL mode. If a chat ever reaches thousands of filters, an index-based approach or caching would be needed.
*Mitigation:* Monitor query times. If needed, add an in-memory trigger cache per chat in a future change.

**[Filesystem I/O inside DB transaction]** Deleting media files inside the SQLite write transaction blocks other writers for the duration of the file deletion. For local SSD storage this is sub-millisecond, but on slow or networked filesystems it could become noticeable.
*Mitigation:* The bot's concurrency is inherently low (single process, message-by-message processing). If this becomes an issue, switch to a deferred garbage collector that periodically sweeps unreferenced media files.

**[No down migrations]** Per project convention, only forward migrations are supported. If the schema needs fixing, a new forward migration must be written. This is consistent with the existing goose setup.
*Mitigation:* Careful schema design upfront (this document). The table-per-type approach is additive — new response types can be added as new tables without modifying existing ones.

**[Afero dependency]** Adding Afero introduces a new dependency. However, it's from the spf13 ecosystem (same author as Viper/Cobra already in use), is widely adopted, and has a stable API.
*Mitigation:* The dependency is isolated behind the `MediaStorage` port interface. If Afero is ever dropped, only the adapter implementation changes.
