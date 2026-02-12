## Why

There is no codebase yet. We need to bootstrap the Patrizio bot project -- a Delta Chat bot focused on group chats that responds to messages based on keyword filters. This initial change establishes the Go project structure, dependencies, tooling, and a minimal working bot that can receive and handle messages, providing the foundation for the filter system and other features to be built on top of.

## What Changes

- Initialize a Go module for the `patrizio` project (latest stable Go version)
- Add `deltabot-cli-go` (bot CLI framework), `rpc-client-go` (Delta Chat RPC bindings), `cobra` (CLI), and `viper` (config) as dependencies
- Create a main entrypoint that wires up `botcli.BotCli` with `OnBotInit` and `OnBotStart` hooks
- Implement basic incoming message handling via `bot.OnNewMsg` callback
- Set up project directory structure following standard Go conventions
- Bot operates in group chats; DMs receive a help/usage response only
- Set up SQLite database with a migration system and separate `.sql` query files (one per query)
- Add Dockerfile with multi-stage build targeting Google distroless images, running as non-root
- Add Makefile with targets for build, run, test, docker-build, migrate, etc.
- Add pre-commit hooks (golangci-lint, go build, docker build, tests on push)
- Add golangci-lint configuration with comprehensive linter set
- Add `.gitignore` for Go artifacts, SQLite files, and build outputs
- Add `README.md` with project overview, setup instructions, usage, and development guide
- Add `LICENSE` file (AGPL-3.0-or-later)

## Capabilities

### New Capabilities

- `bot-skeleton`: Core bot lifecycle -- initialization, startup, shutdown, and basic message reception using deltabot-cli-go. Includes Go module setup, cobra CLI, viper config, and standard project layout.
- `message-handling`: Incoming message routing -- receive messages, distinguish group vs. DM context, and dispatch accordingly. Groups get processed for future filter matching; DMs get a static help response.
- `database`: SQLite database setup with versioned migrations (up/down) and query files kept separate from Go code (one `.sql` file per query).
- `deployment`: Dockerfile using multi-stage build with Google distroless runtime image, non-root execution. Makefile for common dev tasks. Pre-commit hooks and golangci-lint for code quality. README and AGPL-3.0+ license.

### Modified Capabilities

_(none -- greenfield project)_

## Impact

- **Code**: New Go project from scratch -- `main.go`, `go.mod`, `go.sum`, supporting packages, SQL migration and query files
- **Dependencies**: `github.com/deltachat-bot/deltabot-cli-go`, `github.com/chatmail/rpc-client-go`, `github.com/spf13/cobra`, `github.com/spf13/viper`, SQLite driver, migration library
- **Tooling**: `golangci-lint`, `pre-commit`, `make`, `docker`
- **Runtime**: Requires `deltachat-rpc-server` in `PATH`
- **Systems**: No external services beyond the Delta Chat account the bot is initialized with (`patrizio init <email> <password>`)
