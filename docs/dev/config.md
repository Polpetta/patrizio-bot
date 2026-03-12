# Configuration

Patrizio uses **Viper** to load configuration from environment variables, a config file, or hard‑coded defaults.
The three most important variables are:

| Variable | What it controls | Default |
|----------|---|---------|
| `PATRIZIO_DB_PATH` | Path to the SQLite database file | `data/patrizio.db` |
| `PATRIZIO_LOG_LEVEL` | Logging verbosity (`debug`, `info`, `error`) | `info` |
| `PATRIZIO_MEDIA_PATH` | Directory where media files are stored | `data/media` |

The logic lives in `internal/config/config.go`:

```go
func Load() (*Config, error) {
    v := viper.New()
    v.SetEnvPrefix("PATRIZIO")
    v.AutomaticEnv()
    // defaults...
}
```

During boot‑up (`main.go`) these values are used to create the media
directory (`cfg.MediaPath()`). They are also used to open the database
(`cfg.DBPath()`).

---

## File references

* `internal/config/config.go`
* `cmd/patrizio/main.go`
