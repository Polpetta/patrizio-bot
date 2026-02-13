## ADDED Requirements

### Requirement: TOML configuration file support
The system SHALL support loading configuration from a TOML file in addition to environment variables. Environment variables (`PATRIZIO_` prefix) SHALL take priority over TOML values. TOML values SHALL take priority over defaults.

#### Scenario: Config loaded from TOML file
- **WHEN** a TOML config file exists with `media_path = "/data/media"`
- **THEN** the application uses `/data/media` as the media storage path

#### Scenario: Env var overrides TOML
- **WHEN** the TOML config file sets `media_path = "/data/media"` and the env var `PATRIZIO_MEDIA_PATH` is set to `/opt/media`
- **THEN** the application uses `/opt/media` as the media storage path

#### Scenario: Default used when neither set
- **WHEN** neither TOML config nor env var specifies `media_path`
- **THEN** the application uses the default value `./media`

### Requirement: Media path configuration
The system SHALL support a `media_path` configuration key that specifies the directory where media files are stored. The default value SHALL be `./media`.

#### Scenario: Custom media path
- **WHEN** the configuration sets `media_path` to `/var/patrizio/media`
- **THEN** media files are stored in `/var/patrizio/media/`

### Requirement: Dockerfile volume directives
The Dockerfile SHALL declare `VOLUME` directives for data directories that need to persist outside the container. This SHALL include the database storage directory and the media storage directory.

#### Scenario: Data volume declared
- **WHEN** a developer inspects the Dockerfile
- **THEN** a `VOLUME` directive exists for the data directory (containing the database and media)

#### Scenario: Container started with volume mount
- **WHEN** a user runs the container with `-v /host/data:/data`
- **THEN** the database and media files are persisted on the host filesystem

## MODIFIED Requirements

### Requirement: Configuration loading
The system SHALL use Viper to load configuration from environment variables prefixed with `PATRIZIO_`, a TOML config file, and built-in defaults. Priority order: env vars > TOML file > defaults. The config SHALL be loaded into a `Config` struct that implements the domain `Config` port interface, rather than using global Viper state.

#### Scenario: Database path from environment variable
- **WHEN** the environment variable `PATRIZIO_DB_PATH` is set to `/data/bot.db`
- **THEN** the application uses `/data/bot.db` as the SQLite database path

#### Scenario: Default configuration values
- **WHEN** no configuration is provided
- **THEN** the database path defaults to `./patrizio.db`, the log level defaults to `info`, and the media path defaults to `./media`

#### Scenario: Config struct injectable for testing
- **WHEN** tests need a `Config` instance
- **THEN** a `Config` struct can be constructed directly with test values without Viper or file I/O

### Requirement: Bot lifecycle wiring
The system SHALL create a `botcli.BotCli` instance named "patrizio" and register `OnBotInit` and `OnBotStart` hooks before calling `cli.Start()`. The `Setup` function SHALL accept a `Dependencies` struct (or the components needed to build one) so that adapters are injected rather than created internally.

#### Scenario: Bot initializes with Delta Chat account
- **WHEN** a user runs `patrizio init bot@example.com PASSWORD`
- **THEN** the bot configures a Delta Chat account with the provided email and password

#### Scenario: Bot starts and listens for messages
- **WHEN** a user runs `patrizio serve` after initialization
- **THEN** the bot starts, connects to Delta Chat, initializes the database, builds the `Dependencies` struct, and begins processing incoming messages with access to all port implementations

#### Scenario: Bot provides invite link
- **WHEN** a user runs `patrizio link`
- **THEN** the bot outputs its Delta Chat invite link to stdout
