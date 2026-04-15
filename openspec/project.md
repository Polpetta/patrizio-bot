# Project Context

## Purpose
Patrizio is a Delta Chat bot written in Go that responds to incoming messages in group chats based on keyword filters. It can reply with text, reactions, or media (stickers, GIFs, images, videos). The filter system is inspired by Miss Rose on Telegram.

## Tech Stack
- **Language:** Go (1.25+)
- **Bot framework:** `github.com/deltachat-bot/deltabot-cli-go/v2/botcli` -- CLI scaffolding, bot lifecycle hooks (`OnBotInit`, `OnBotStart`), and logging
- **Delta Chat client:** `github.com/chatmail/rpc-client-go/v2/deltachat` -- Go bindings for the Delta Chat RPC API
- **CLI framework:** `github.com/spf13/cobra` (used transitively via deltabot-cli-go)
- **Configuration:** `github.com/spf13/viper` -- env vars prefixed `PATRIZIO_`
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Migrations:** `github.com/pressly/goose/v3` -- forward-only, no down migrations
- **SQL queries:** `sqlc` -- generates type-safe Go from `.sql` files
- **Linting:** `golangci-lint` with `.golangci.yml` config
- **Pre-commit hooks:** `.pre-commit-config.yaml`
- **Docker:** Multi-stage build, `gcr.io/distroless/static-debian12`, non-root
- **Runtime dependency:** `deltachat-rpc-server` (bundled in Docker image from GitHub releases)

## Project Conventions

### Code Style
- Standard Go formatting (`gofmt`/`goimports`)
- All packages must have package-level doc comments
- All errors must be checked (`errcheck` linter enabled)
- Unused parameters should be named `_`
- Import grouping: stdlib, external, internal (`github.com/polpetta/patrizio`)

### Architecture Patterns
- **Project layout:** `cmd/patrizio/` for entrypoint, `internal/` for all private packages
- **Bot lifecycle:** `OnBotInit` registers message handlers (runs for all commands), `OnBotStart` initializes DB and runs migrations (runs only for `serve`)
- **Database:** SQL queries live in `queries/*.sql`, generated Go code in `internal/database/queries/`. No inline SQL in Go files.
- **Migrations:** Forward-only `.up.sql` files in `migrations/`, embedded via `go:embed` at build time
- **Configuration:** Viper loads from env vars (`PATRIZIO_DB_PATH`, `PATRIZIO_LOG_LEVEL`) with sensible defaults

### Testing Strategy
- `go test ./...` via `make test`
- Test files have relaxed lint rules (no `gocyclo`, `errcheck`, `gosec`, `funlen`)

### Git Workflow
- Pre-commit hooks: `golangci-lint run --fix`, `go build ./...`, `make docker-build`
- Pre-push hooks: `make test`

## Domain Context
- **Delta Chat** is a decentralized messaging app that uses email infrastructure (IMAP/SMTP). Bots interact via the Delta Chat RPC API.
- **Bot init flow:** `patrizio init email password` -> `patrizio link` -> `patrizio serve`
- **Message routing:** Special contacts (system/device messages) are ignored. Group messages go to the filter engine (placeholder). DMs get a static help response.
- **Filter system (future):** Inspired by Miss Rose on Telegram -- single-word filters, multi-word phrase filters, prefix filters, exact filters, media responses, runtime CRUD management.

## Important Constraints
- **No CGO:** Pure Go SQLite driver (`modernc.org/sqlite`) is required for distroless Docker compatibility
- **No down migrations:** Only forward migrations with goose
- **License:** AGPL-3.0-or-later
- **`deltachat-rpc-server` must be in PATH** at runtime

## External Dependencies
- **Delta Chat RPC server:** `deltachat-rpc-server` binary from [chatmail/core](https://github.com/chatmail/core/releases) -- must be available in PATH. Bundled in the Docker image.
- **Delta Chat account:** Requires an email account (e.g. from chatmail) for the bot to operate
