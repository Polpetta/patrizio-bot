## MODIFIED Requirements

### Requirement: Configuration loading
The system SHALL use viper to load configuration from environment variables prefixed with `PATRIZIO_` and an optional config file. The configuration SHALL include the following keys for OpenAI integration:
- `openai_base_url`: The base URL of the OpenAI-compatible API endpoint, including the `/v1` path segment (e.g. `https://api.openai.com/v1`). No default — feature disabled if unset
- `openai_api_key`: The API key for authentication (no default — feature disabled if unset)
- `openai_model`: The model identifier to use for chat completions (default: `gpt-4o-mini`)
- `openai_max_history`: Maximum number of conversation messages per thread to send as context (default: `50`)
- `openai_allowed_chat_ids`: List of Delta Chat chat IDs allowed to use `/prompt` (default: empty — all chats allowed)
- `openai_system_prompt`: System prompt prepended to every API call as a system/developer message (default: `You are a helpful assistant.`). When set to a non-empty value, this message is included as the first message in every chat completion request.

#### Scenario: Database path from environment variable
- **WHEN** the environment variable `PATRIZIO_DB_PATH` is set to `/data/bot.db`
- **THEN** the application uses `/data/bot.db` as the SQLite database path

#### Scenario: Default configuration values
- **WHEN** no configuration is provided
- **THEN** the database path defaults to `./patrizio.db`, the log level defaults to `info`, `openai_model` defaults to `gpt-4o-mini`, `openai_max_history` defaults to `50`, `openai_allowed_chat_ids` defaults to an empty list, and `openai_system_prompt` defaults to `You are a helpful assistant.`

#### Scenario: OpenAI configuration from environment
- **WHEN** `PATRIZIO_OPENAI_BASE_URL` is set to `https://api.openai.com/v1` and `PATRIZIO_OPENAI_API_KEY` is set to `sk-xxx`
- **THEN** the application uses these values for the AI client configuration

#### Scenario: OpenAI configuration from TOML
- **WHEN** `patrizio.toml` contains `openai_base_url = "http://localhost:11434/v1"` and `openai_api_key = "ollama"`
- **THEN** the application uses these values for the AI client configuration

### Requirement: Bot lifecycle wiring
The system SHALL create a `botcli.BotCli` instance named "patrizio" and register `OnBotInit` and `OnBotStart` hooks before calling `cli.Start()`. During dependency construction, the system SHALL create an `AIClient` instance using the OpenAI configuration values and include it in the `Dependencies` struct.

#### Scenario: Bot initializes with Delta Chat account
- **WHEN** a user runs `patrizio init bot@example.com PASSWORD`
- **THEN** the bot configures a Delta Chat account with the provided email and password

#### Scenario: Bot starts and listens for messages
- **WHEN** a user runs `patrizio serve` after initialization
- **THEN** the bot starts, connects to Delta Chat, and begins processing incoming messages with the AI client available for `/prompt` commands

#### Scenario: Bot provides invite link
- **WHEN** a user runs `patrizio link`
- **THEN** the bot outputs its Delta Chat invite link to stdout

#### Scenario: AI client not created when unconfigured
- **WHEN** the bot starts without `openai_base_url` or `openai_api_key` configured
- **THEN** the `AIClient` field in Dependencies is nil, and `/prompt` commands reply with a configuration error
