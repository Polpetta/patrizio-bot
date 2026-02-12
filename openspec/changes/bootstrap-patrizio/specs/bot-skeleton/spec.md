## ADDED Requirements

### Requirement: Go module initialization
The project SHALL be initialized as a Go module with the latest stable Go version. The module path SHALL follow standard Go conventions.

#### Scenario: Module is valid
- **WHEN** a developer clones the repository and runs `go mod tidy`
- **THEN** all dependencies resolve successfully and the module is ready for building

### Requirement: Project directory structure
The project SHALL follow standard Go project layout with `cmd/` for entrypoints and `internal/` for private packages.

#### Scenario: Entrypoint exists at standard location
- **WHEN** a developer inspects the project structure
- **THEN** `cmd/patrizio/main.go` exists as the binary entrypoint

#### Scenario: Internal packages are not importable externally
- **WHEN** an external Go module attempts to import packages under `internal/`
- **THEN** the Go compiler rejects the import

### Requirement: Bot lifecycle wiring
The system SHALL create a `botcli.BotCli` instance named "patrizio" and register `OnBotInit` and `OnBotStart` hooks before calling `cli.Start()`.

#### Scenario: Bot initializes with Delta Chat account
- **WHEN** a user runs `patrizio init bot@example.com PASSWORD`
- **THEN** the bot configures a Delta Chat account with the provided email and password

#### Scenario: Bot starts and listens for messages
- **WHEN** a user runs `patrizio serve` after initialization
- **THEN** the bot starts, connects to Delta Chat, and begins processing incoming messages

#### Scenario: Bot provides invite link
- **WHEN** a user runs `patrizio link`
- **THEN** the bot outputs its Delta Chat invite link to stdout

### Requirement: Configuration loading
The system SHALL use viper to load configuration from environment variables prefixed with `PATRIZIO_` and an optional config file.

#### Scenario: Database path from environment variable
- **WHEN** the environment variable `PATRIZIO_DB_PATH` is set to `/data/bot.db`
- **THEN** the application uses `/data/bot.db` as the SQLite database path

#### Scenario: Default configuration values
- **WHEN** no configuration is provided
- **THEN** the database path defaults to `./patrizio.db` and the log level defaults to `info`

### Requirement: Bot data storage location
The bot SHALL store its Delta Chat account data in a platform-specific user config directory by default, overridable with the `--folder` flag.

#### Scenario: Custom data folder
- **WHEN** a user runs `patrizio --folder /custom/path serve`
- **THEN** the bot stores its Delta Chat data in `/custom/path`

#### Scenario: Default data folder
- **WHEN** a user runs `patrizio serve` without `--folder`
- **THEN** the bot stores its Delta Chat data in the platform default directory (e.g. `~/.config/patrizio/` on Linux)
