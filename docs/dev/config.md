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
```

Note that this file can be found in the root repository of the project as well.

## Environment Variable Overrides

Each key can be overridden by an environment variable prefixed with `PATRIZIO_`. Examples:

```bash
export PATRIZIO_DB_PATH="/tmp/patrizio.db"
export PATRIZIO_LOG_LEVEL="warn"
export PATRIZIO_MEDIA_PATH="/tmp/media"
```

Environment variables take precedence over values in the TOML file.
