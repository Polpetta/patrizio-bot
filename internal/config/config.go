// Package config provides application configuration via environment variables.
package config

import (
	"github.com/spf13/viper"
)

const (
	// KeyDBPath is the configuration key for the SQLite database file path.
	KeyDBPath = "db_path"
	// KeyLogLevel is the configuration key for the log level.
	KeyLogLevel = "log_level"
)

// Load initializes viper with defaults and environment variable bindings.
// Environment variables are prefixed with PATRIZIO_ (e.g. PATRIZIO_DB_PATH).
func Load() {
	viper.SetEnvPrefix("PATRIZIO")
	viper.AutomaticEnv()

	viper.SetDefault(KeyDBPath, "./patrizio.db")
	viper.SetDefault(KeyLogLevel, "info")
}

// DBPath returns the configured SQLite database file path.
func DBPath() string {
	return viper.GetString(KeyDBPath)
}

// LogLevel returns the configured log level.
func LogLevel() string {
	return viper.GetString(KeyLogLevel)
}
