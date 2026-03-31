---
icon: lucide/file-sliders
---

# Configuration

Patrizio loads configuration from three sources in this precedence order:

1. **Environment variables** – prefixed with `PATRIZIO_` (e.g. `PATRIZIO_DB_PATH`). Override everything else.
2. **TOML config file** – by default named `patrizio.toml`. Place it in the
current directory or `/etc/patrizio/` to persist settings.
3. **Default values** – set in the application when no file or env var is provided.

The following table lists configuration keys, types, and defaults:

| Key          | Type   | Default                | Description                                          |
|--------------|--------|------------------------|------------------------------------------------------|
| `db_path`    | string | `/data/db/patrizio.db` | SQLite database file path                            |
| `log_level`  | string | `info`                 | Logging verbosity (`debug`, `info`, `warn`, `error`) |
| `media_path` | string | `/data/media`          | Directory where media files are stored               |

## OpenAI / AI Prompt keys

These keys configure the `/prompt` command. The feature is disabled unless `openai_api_key` is set.

| Key                       | Type     | Default                        | Description                                                        |
|---------------------------|----------|--------------------------------|--------------------------------------------------------------------|
| `openai_api_key`          | string   | _(empty)_                      | API key for the OpenAI-compatible provider. Required to enable AI. |
| `openai_base_url`         | string   | _(empty)_                      | Custom base URL (for Ollama, LMStudio, etc.). Empty = OpenAI.      |
| `openai_model`            | string   | `gpt-4o-mini`                  | Model identifier sent to the API.                                  |
| `openai_max_history`      | int      | `50`                           | Max conversation messages sent as context per thread.              |
| `openai_system_prompt`    | string   | `You are a helpful assistant.` | System prompt prepended to every request.                          |
| `openai_allowed_chat_ids` | int list | _(empty)_                      | Chat ID allowlist. Empty = all chats allowed.                      |

## TOML File Example

```toml
# patrizio.toml

# SQLite database file
# Default: /data/db/patrizio.db
db_path = "/var/lib/patrizio/patrizio.db"

# Logging level – one of: debug, info, warn, error
log_level = "debug"

# Directory where media files are stored
# Default: /data/media
media_path = "/var/lib/patrizio/media"

# OpenAI-compatible API configuration (for /prompt command)
# API key (required to enable the /prompt command)
#openai_api_key = ""

# Base URL for OpenAI-compatible API (optional, defaults to OpenAI's API)
#openai_base_url = ""

# Model to use for chat completions (default: "gpt-4o-mini")
#openai_model = "gpt-4o-mini"

# Maximum number of conversation history messages to include (default: 50)
#openai_max_history = 50

# System prompt prepended to every conversation (default: "You are a helpful assistant.")
#openai_system_prompt = "You are a helpful assistant."

# Chat ID allowlist — if non-empty, only these chats can use /prompt (default: empty = all allowed)
#openai_allowed_chat_ids = []
```

Note that this file can be found in the root repository of the project as well.

## Environment Variable Overrides

Each key can be overridden by an environment variable prefixed with `PATRIZIO_`. Examples:

```bash
export PATRIZIO_DB_PATH="/tmp/patrizio.db"
export PATRIZIO_LOG_LEVEL="warn"
export PATRIZIO_MEDIA_PATH="/tmp/media"

# OpenAI configuration (recommended to use env vars for the API key)
export PATRIZIO_OPENAI_API_KEY="sk-..."
export PATRIZIO_OPENAI_MODEL="gpt-4o-mini"
```

Environment variables take precedence over values in the TOML file.
