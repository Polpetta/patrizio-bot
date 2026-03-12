# Architecture Overview

Patrizio is a lightweight Delta Chat bot written in pure Go (1.25+).
The code is split into logical layers, each with a single responsibility.

* `cmd/patrizio/main.go` boots the application:
  * loads configuration
  * creates media and database directories
  * opens the SQLite database
  * builds the adapters
  * hands everything to the bot framework

* `internal/bot/bot.go` registers a `OnNewMsg` callback with the `deltabot‑cli‑go` library.
  All the real work – parsing commands, looking up filters, and replying – happens in `handler.go`.

* `internal/domain/filter.go` normalises triggers; `internal/domain/command.go` parses commands.
  The ports that the adapters implement are defined here.
  All functions are pure and side‑effect‑free, making them easy to test.

* `internal/adapter/sqlite/repository.go` wraps a `*sql.DB` and forwards to the `sqlc`‑generated queries.
  `internal/adapter/storage/storage.go` reads and writes media files to the directory defined by configuration.

The message handling flow is straightforward:

```
User → Delta Chat RPC → BotCli (deltabot‑cli‑go) →
OnNewMsg → handler.go → domain logic → repository → reply via RPC
```

`handler.go` decides whether a message is a command, a direct message, or a group message.
  Commands are parsed by the domain code.
  The repository is queried for matching filters.
  The bot replies with text, media, or reactions.

The design choices are also documented in the `openspec/` folder.
It contains detailed change proposals and architecture discussions.

---

## References

* `cmd/patrizio/main.go`
* `internal/bot/bot.go`
* `internal/bot/handler.go`
* `internal/domain/filter.go`
* `internal/domain/command.go`
* `internal/adapter/sqlite/repository.go`
* `internal/adapter/storage/storage.go`
* `internal/database/queries/*`
