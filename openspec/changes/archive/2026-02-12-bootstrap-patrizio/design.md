## Context

Patrizio is a greenfield Delta Chat bot written in Go, designed for group chats. There is no existing codebase. The bot framework is `deltabot-cli-go`, which provides CLI scaffolding (via cobra), bot lifecycle hooks (`OnBotInit`, `OnBotStart`), and logging. The Delta Chat RPC API is accessed through `github.com/chatmail/rpc-client-go/deltachat`. The runtime requires `deltachat-rpc-server` in `PATH`.

This design covers the initial bootstrap: a working bot skeleton with message routing, SQLite persistence layer, Docker deployment, and developer tooling.

## Goals / Non-Goals

**Goals:**

- Establish a clean, idiomatic Go project structure that supports incremental feature development
- Get a bot that starts, connects to Delta Chat, receives messages, and routes them (group vs. DM)
- Set up SQLite with a migration framework and separated SQL query files from day one
- Provide a containerized deployment path (distroless, non-root)
- Enforce code quality from the start (linting, pre-commit hooks)

**Non-Goals:**

- Filter/keyword engine (future change)
- Media response handling (future change)
- Filter CRUD commands (future change)
- CI/CD pipelines (future change)
- Advanced configuration management beyond basic viper setup
- Multi-account bot support

## Decisions

### 1. Project Layout

**Decision**: Follow the standard Go project layout convention with `cmd/` for entrypoints and `internal/` for private packages.

```
patrizio/
├── cmd/
│   └── patrizio/
│       └── main.go              # Entrypoint
├── internal/
│   ├── bot/
│   │   ├── bot.go               # Bot setup, lifecycle, OnBotInit/OnBotStart wiring
│   │   └── handler.go           # Message handler (group dispatch + DM help)
│   ├── config/
│   │   └── config.go            # Viper config loading
│   └── database/
│       ├── database.go          # SQLite connection, migration runner
│       └── queries/             # Generated or hand-written query functions
├── migrations/
│   └── 001_initial.up.sql
├── queries/
│   └── example.sql              # One .sql file per query
├── Dockerfile
├── Makefile
├── .pre-commit-config.yaml
├── .golangci.yml
├── .gitignore
├── README.md
├── LICENSE
├── go.mod
└── go.sum
```

**Rationale**: `cmd/` + `internal/` is the de-facto standard for Go projects. It separates the binary entrypoint from library code and prevents external imports of internal packages. The `migrations/` and `queries/` directories live at the project root for visibility and easy tooling integration.

### 2. SQL Query Management

**Decision**: Use `sqlc` to generate type-safe Go code from `.sql` query files.

**Rationale**: `sqlc` is purpose-built for this -- you write SQL in `.sql` files (one per query or grouped by domain), and it generates idiomatic Go functions with proper types. This satisfies the requirement of keeping SQL separate from Go code while providing compile-time safety. The alternative would be hand-writing query functions that read `.sql` files at runtime, but that loses type safety and adds I/O overhead.

**Alternatives considered**:
- **Hand-written query loader** (read `.sql` files at runtime): Simpler but no type safety, runtime file I/O, error-prone.
- **Inline SQL strings**: Explicitly ruled out by requirements.
- **GORM / ent**: ORM approach hides SQL, doesn't meet the "proper SQL files" requirement.

### 3. Migration Framework

**Decision**: Use `goose` for forward-only database migrations (up only, no down migrations).

**Rationale**: `goose` supports both SQL and Go migration files, has a clean CLI, integrates well with `sqlc`, and supports SQLite. It can be configured for forward-only migrations by simply omitting down files. It can also be embedded into the application for programmatic migration execution at startup. Down migrations are not needed -- if a migration is wrong, a new forward migration corrects it.

**Alternatives considered**:
- **golang-migrate**: Also solid, but `goose` has better ergonomics for embedding migrations into the binary and running them programmatically.
- **Manual schema versioning**: Error-prone, no automation.

### 4. SQLite Driver

**Decision**: Use `modernc.org/sqlite` (pure Go SQLite implementation).

**Rationale**: Pure Go means no CGO dependency, which simplifies cross-compilation and Docker builds (especially with distroless images that have no C runtime). Performance is adequate for a bot's workload.

**Alternatives considered**:
- **mattn/go-sqlite3**: Most popular, but requires CGO. Complicates Docker multi-stage builds with distroless (need to statically link or include C libs).

### 5. Docker Image Strategy

**Decision**: Multi-stage build with `golang:1.24` builder and `gcr.io/distroless/static-debian12` runtime.

```dockerfile
# Builder
FROM golang:1.24 AS builder
# ... build steps ...

# Runtime
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/patrizio /patrizio
USER nonroot:nonroot
ENTRYPOINT ["/patrizio"]
```

**Rationale**: `static-debian12` is the smallest distroless variant (~2MB). Since we use a pure Go SQLite driver (no CGO), the binary is fully static and doesn't need glibc. Running as `nonroot` is built into distroless images (uid 65534).

### 6. Bot Message Routing

**Decision**: Single `OnNewMsg` handler that checks chat type and dispatches:
- **Group messages**: Log receipt, pass through (no-op for now, future filter engine hooks here)
- **DM messages**: Reply with static help/usage text

**Rationale**: Keep it simple for the skeleton. The handler structure should make it easy to plug in the filter engine later without restructuring.

### 7. Bot Initialization and Identity

**Decision**: Follow the standard deltabot-cli-go initialization flow. The bot is configured with an email address and password via the `init` subcommand, then started with `serve`. A `link` subcommand provides the bot's invite link for users to contact it.

```sh
# Configure the bot's Delta Chat account
patrizio init bot@example.com PASSWORD

# Get invite link to share with users
patrizio link

# Start the bot
patrizio serve
```

Bot data is stored in a platform-specific user config directory by default (e.g. `~/.config/patrizio/` on Linux), overridable with `--folder PATH`. This is handled by `deltabot-cli-go` -- the `init`, `serve`, `link`, and `--folder` flag all come from the framework out of the box.

**Rationale**: This is the convention established by deltabot-cli-go. No reason to deviate. The email/password are stored by the Delta Chat RPC layer in its own database, separate from our SQLite application database.

### 8. Configuration with Viper

**Decision**: Use viper for configuration, supporting environment variables and an optional config file. Minimal config for now:
- `PATRIZIO_DB_PATH` -- SQLite database file path (default: `./patrizio.db`)
- `PATRIZIO_LOG_LEVEL` -- Log level (default: `info`)

**Rationale**: Viper pairs naturally with cobra (both from spf13), supports env vars out of the box (good for Docker), and config files for local dev. We keep the config surface small for the bootstrap and expand as features are added.

### 9. Makefile Targets

**Decision**: Provide these targets:

| Target | Description |
|--------|-------------|
| `build` | `go build ./cmd/patrizio` |
| `run` | `go run ./cmd/patrizio serve` |
| `test` | `go test ./...` |
| `lint` | `golangci-lint run` |
| `docker-build` | Multi-stage Docker build |
| `migrate` | Run pending migrations |
| `migrate-create` | Create a new migration file |
| `sqlc` | Regenerate query code from `.sql` files |
| `clean` | Remove build artifacts |

## Risks / Trade-offs

**[Pure Go SQLite may be slower than CGO version]** → Acceptable for a bot workload. If performance becomes an issue in the future, switching to `mattn/go-sqlite3` is straightforward since both use `database/sql` interface.

**[sqlc adds a code generation step]** → Developers must run `make sqlc` after changing query files. Mitigated by documenting this in the README and potentially adding it to pre-commit hooks later.

**[goose migrations embedded in binary]** → Migration files must be embedded via `go:embed`. This is standard practice and means the binary is self-contained for deployment.

**[Distroless images have no shell for debugging]** → Intentional trade-off for security. For debugging, developers can use `docker exec` with a debug variant or attach to the running process.

**[deltabot-cli-go controls the CLI structure]** → The framework owns the top-level cobra command (`init`, `serve`). Custom commands must integrate within its structure. This limits flexibility but provides battle-tested bot lifecycle management.
