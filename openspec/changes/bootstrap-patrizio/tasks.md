## 1. Project Initialization & Tooling

- [ ] 1.1 Initialize Go module (`go mod init`) with latest stable Go version
- [ ] 1.2 Create project directory structure: `cmd/patrizio/`, `internal/bot/`, `internal/config/`, `internal/database/`, `migrations/`, `queries/`
- [ ] 1.3 Add `.gitignore` for Go build artifacts, SQLite `.db` files, IDE configs, and generated files
- [ ] 1.4 Add `.golangci.yml` with the provided comprehensive linter configuration (update `goimports.local-prefixes` to match the project module path)
- [ ] 1.5 Add `.pre-commit-config.yaml` with the provided hook configuration (lint, build, docker-build on commit; tests on push)
- [ ] 1.6 Add `LICENSE` file with AGPL-3.0-or-later full text

## 2. Makefile

- [ ] 2.1 Create `Makefile` with targets: `build`, `run`, `test`, `lint`, `docker-build`, `migrate`, `migrate-create`, `sqlc`, `clean`

## 3. Configuration

- [ ] 3.1 Create `internal/config/config.go` with viper setup: env var prefix `PATRIZIO_`, config keys for `DB_PATH` (default `./patrizio.db`) and `LOG_LEVEL` (default `info`)

## 4. Database Layer

- [ ] 4.1 Create `internal/database/database.go` with SQLite connection setup using `modernc.org/sqlite` via `database/sql`
- [ ] 4.2 Integrate `goose` for forward-only migrations with `go:embed` for the `migrations/` directory
- [ ] 4.3 Create `migrations/001_initial.up.sql` with a minimal placeholder table to verify the migration system works
- [ ] 4.4 Add `sqlc.yaml` configuration file for code generation from `queries/` to `internal/database/queries/`
- [ ] 4.5 Create a placeholder `.sql` query file in `queries/` and run `sqlc generate` to produce initial Go query code

## 5. Bot Skeleton

- [ ] 5.1 Create `internal/bot/bot.go` with `botcli.New("patrizio")` setup, `OnBotInit` and `OnBotStart` hook registration, and database initialization on start
- [ ] 5.2 Create `cmd/patrizio/main.go` entrypoint that wires config, database, and bot together, then calls `cli.Start()`

## 6. Message Handling

- [ ] 6.1 Create `internal/bot/handler.go` with the `OnNewMsg` callback registered during `OnBotInit`
- [ ] 6.2 Implement special contact filtering: skip messages where `FromId <= deltachat.ContactLastSpecial`
- [ ] 6.3 Implement chat type detection using Delta Chat RPC API to distinguish group vs. DM
- [ ] 6.4 Implement group message routing: log receipt, no-op placeholder for future filter engine
- [ ] 6.5 Implement DM help response: reply with static help/usage text including bot name, purpose, and instructions to add to a group

## 7. Docker

- [ ] 7.1 Create `Dockerfile` with multi-stage build: `golang:1.24` builder, `gcr.io/distroless/static-debian12` runtime, `USER nonroot:nonroot`, static binary compilation (CGO_ENABLED=0)

## 8. Documentation

- [ ] 8.1 Create `README.md` with project overview, prerequisites (`deltachat-rpc-server`, Go, Docker), setup instructions, bot initialization flow (`init`, `link`, `serve`), Makefile usage, and development guide (pre-commit, linting, migrations, sqlc)

## 9. Verification

- [ ] 9.1 Run `go mod tidy` and verify all dependencies resolve
- [ ] 9.2 Run `make build` and verify the binary compiles
- [ ] 9.3 Run `make lint` and fix any linter issues
- [ ] 9.4 Run `make docker-build` and verify the Docker image builds
- [ ] 9.5 Run `make test` and verify tests pass (even if no tests yet, the command should succeed)
