# Patrizio

A Delta Chat bot for group chats, built with Go. Patrizio responds to messages based on configured keyword filters, inspired by [Miss Rose](https://missrose.org/) on Telegram.

## Prerequisites

- **Go** 1.24+
- **deltachat-rpc-server** — must be available in `PATH`. See [installation instructions](https://github.com/chatmail/core/tree/master/deltachat-rpc-server).
- **Docker** (optional, for containerized deployment)
- **golangci-lint** (for development)
- **sqlc** (for regenerating query code)
- **goose** (for creating new migrations)
- **pre-commit** (for git hooks)

## Quick Start

### 1. Build

```sh
make build
```

### 2. Initialize the bot

Configure the bot with a Delta Chat email account:

```sh
./patrizio init bot@example.com YOUR_PASSWORD
```

### 3. Get the invite link

Share this link so users can contact the bot:

```sh
./patrizio link
```

### 4. Run the bot

```sh
./patrizio serve
```

The bot will connect to Delta Chat and start processing messages. Add it to a group to use keyword filters, or send it a direct message to get help text.

## Configuration

Configuration is done via environment variables prefixed with `PATRIZIO_`:

| Variable | Default | Description |
|---|---|---|
| `PATRIZIO_DB_PATH` | `./patrizio.db` | Path to the SQLite database file |
| `PATRIZIO_LOG_LEVEL` | `info` | Log level |

The bot's Delta Chat account data is stored in a platform-specific config directory (e.g. `~/.config/patrizio/` on Linux), overridable with `--folder`:

```sh
./patrizio --folder /custom/path serve
```

## Docker

Build and run with Docker:

```sh
make docker-build
docker run -v patrizio-data:/data -e PATRIZIO_DB_PATH=/data/patrizio.db patrizio serve
```

The Docker image uses `gcr.io/distroless/static-debian12` and runs as a non-root user.

## Development

### Setup

Install pre-commit hooks:

```sh
pre-commit install
pre-commit install --hook-type pre-push
```

### Makefile Targets

| Target | Description |
|---|---|
| `make build` | Compile the binary |
| `make run` | Run the bot in serve mode |
| `make test` | Run all tests |
| `make lint` | Run golangci-lint |
| `make docker-build` | Build the Docker image |
| `make migrate` | Run pending database migrations |
| `make migrate-create NAME=<name>` | Create a new migration file |
| `make sqlc` | Regenerate Go code from SQL query files |
| `make clean` | Remove build artifacts |

### Database Migrations

Migrations are forward-only (no down migrations). To create a new migration:

```sh
make migrate-create NAME=add_filters_table
```

This creates a new `.sql` file in `migrations/`. Edit it, then run:

```sh
make migrate
```

Migrations are also run automatically on bot startup.

### SQL Queries

SQL queries live in `queries/` as `.sql` files. After editing, regenerate the Go code:

```sh
make sqlc
```

Generated code is written to `internal/database/queries/`.

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE) for the full text.
