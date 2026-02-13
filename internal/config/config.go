// Package config provides application configuration via environment variables and TOML files.
package config

import (
	"github.com/spf13/viper"
)

const (
	// KeyDBPath is the configuration key for the SQLite database file path.
	KeyDBPath = "db_path"
	// KeyLogLevel is the configuration key for the log level.
	KeyLogLevel = "log_level"
	// KeyMediaPath is the configuration key for the media storage directory.
	KeyMediaPath = "media_path"
)

// Config holds application configuration values and implements the domain Config port interface.
type Config struct {
	dbPath    string
	logLevel  string
	mediaPath string
}

// DBPath returns the configured SQLite database file path.
func (c *Config) DBPath() string {
	return c.dbPath
}

// LogLevel returns the configured log level.
func (c *Config) LogLevel() string {
	return c.logLevel
}

// MediaPath returns the configured media storage directory path.
func (c *Config) MediaPath() string {
	return c.mediaPath
}

// Load initializes viper with defaults, TOML file, and environment variable bindings.
// Environment variables are prefixed with PATRIZIO_ (e.g. PATRIZIO_DB_PATH) and take priority over TOML.
// Returns a Config struct with the loaded values.
func Load() (*Config, error) {
	viper.SetEnvPrefix("PATRIZIO")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault(KeyDBPath, "/data/db/patrizio.db")
	viper.SetDefault(KeyLogLevel, "info")
	viper.SetDefault(KeyMediaPath, "/data/media")

	// Try to load TOML config file (optional)
	viper.SetConfigName("patrizio")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/patrizio/")

	// Ignore error if config file not found — env vars and defaults still work
	_ = viper.ReadInConfig()

	return &Config{
		dbPath:    viper.GetString(KeyDBPath),
		logLevel:  viper.GetString(KeyLogLevel),
		mediaPath: viper.GetString(KeyMediaPath),
	}, nil
}
