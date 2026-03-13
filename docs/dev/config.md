# Configuration

Patrizio loads configuration from three sources in this precedence order:

1. **Environment variables** – prefixed with `PATRIZIO_` (e.g. `PATRIZIO_DB_PATH`). Override everything else.
2. **TOML config file** – by default named `patrizio.toml`. Place it in the
current directory or `/etc/patrizio/` to persist settings.
3. **Default values** – set in the application when no file or env var is provided.

The following table lists configuration keys, types, and defaults.

| Key          | Type   | Default                | Description |
|---------------|--------|-----------------------|---------------|
| `db_path`    | string | `/data/db/patrizio.db` | SQLite database file path |
| `log_level`  | string | `info`                | Logging verbosity (`debug`, `info`, `warn`, `error`) |
| `media_path` | string | `/data/media`         | Directory where media files are stored |

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

## Environment Variable Overrides

Each key can be overridden by an environment variable prefixed with `PATRIZIO_`. Examples:

```bash
export PATRIZIO_DB_PATH="/tmp/patrizio.db"
export PATRIZIO_LOG_LEVEL="warn"
export PATRIZIO_MEDIA_PATH="/tmp/media"
```

Environment variables take precedence over values in the TOML file.

## Search Paths for the TOML File

The configuration loader looks for `patrizio.toml` in these directories (in order):

1. Current working directory (`.`)
2. `/etc/patrizio/`

If no file is found, the application uses default values instead.

## Consuming the Configuration

The `Load` function in `internal/config/config.go` returns a `Config` instance.

Example usage in `cmd/patrizio/main.go`:

```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("failed to load config: %v", err)
}

// Use the configuration
log.Infof("using media path %s", cfg.MediaPath())
```

The configuration values are used throughout the application.

For example, `cfg.MediaPath()` determines where media files are served from.

## Quick-start Snippet

```bash
# Create a custom configuration file
cat > patrizio.toml <<EOF
# patrizio.toml
log_level = "debug"
EOF

# Run the bot with the custom config
./patrizio
```

## Reference

See the source code: [internal/config/config.go](../internal/config/config.go).
