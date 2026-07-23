// Package config provides application configuration via environment variables and TOML files.
package config

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	// KeyDBPath is the configuration key for the SQLite database file path.
	KeyDBPath = "db_path"
	// KeyLogLevel is the configuration key for the log level.
	KeyLogLevel = "log_level"
	// KeyMediaPath is the configuration key for the media storage directory.
	KeyMediaPath = "media_path"
	// KeyChatStatePath is the configuration key for the per-chat state storage directory.
	KeyChatStatePath = "chat_state_path"
	// KeyOpenAIBaseURL is the configuration key for the OpenAI-compatible API base URL.
	KeyOpenAIBaseURL = "openai_base_url"
	// KeyOpenAIAPIKey is the configuration key for the OpenAI API key.
	KeyOpenAIAPIKey = "openai_api_key" //nolint:gosec // config key name, not a credential
	// KeyOpenAIModel is the configuration key for the OpenAI model identifier.
	KeyOpenAIModel = "openai_model"
	// KeyOpenAIMaxHistory is the configuration key for the max conversation history length.
	KeyOpenAIMaxHistory = "openai_max_history"
	// KeyOpenAIAllowedChatIDs is the configuration key for the chat ID allowlist.
	KeyOpenAIAllowedChatIDs = "openai_allowed_chat_ids"
	// KeyOpenAISystemPrompt is the configuration key for the system prompt.
	KeyOpenAISystemPrompt = "openai_system_prompt"
	// KeyOpenAIMaxToolIterations is the configuration key for the max tool-calling loop iterations per turn.
	KeyOpenAIMaxToolIterations = "openai_max_tool_iterations"
	// KeyOpenAIMaxMemoryBytes is the configuration key for the maximum memory file size in bytes.
	KeyOpenAIMaxMemoryBytes = "openai_max_memory_bytes"
)

// Config holds application configuration values and implements the domain Config port interface.
type Config struct {
	dbPath                  string
	logLevel                string
	mediaPath               string
	chatStatePath           string
	openAIBaseURL           string
	openAIAPIKey            string
	openAIModel             string
	openAIMaxHistory        int
	openAIAllowedChatIDs    []int64
	openAISystemPrompt      string
	openAIMaxToolIterations int
	openAIMaxMemoryBytes    int
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

// ChatStatePath returns the configured per-chat state storage directory path.
func (c *Config) ChatStatePath() string {
	return c.chatStatePath
}

// OpenAIBaseURL returns the configured OpenAI-compatible API base URL.
func (c *Config) OpenAIBaseURL() string {
	return c.openAIBaseURL
}

// OpenAIAPIKey returns the configured OpenAI API key.
func (c *Config) OpenAIAPIKey() string {
	return c.openAIAPIKey
}

// OpenAIModel returns the configured OpenAI model identifier.
func (c *Config) OpenAIModel() string {
	return c.openAIModel
}

// OpenAIMaxHistory returns the maximum number of conversation messages to send as context.
func (c *Config) OpenAIMaxHistory() int {
	return c.openAIMaxHistory
}

// OpenAIAllowedChatIDs returns the list of chat IDs allowed to use /prompt.
func (c *Config) OpenAIAllowedChatIDs() []int64 {
	return c.openAIAllowedChatIDs
}

// OpenAISystemPrompt returns the system prompt prepended to API requests.
func (c *Config) OpenAISystemPrompt() string {
	return c.openAISystemPrompt
}

// OpenAIMaxToolIterations returns the maximum number of tool-calling loop iterations per turn.
func (c *Config) OpenAIMaxToolIterations() int {
	return c.openAIMaxToolIterations
}

// OpenAIMaxMemoryBytes returns the maximum allowed size of a chat memory file in bytes.
func (c *Config) OpenAIMaxMemoryBytes() int {
	return c.openAIMaxMemoryBytes
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
	viper.SetDefault(KeyChatStatePath, "/data/chat_state")
	viper.SetDefault(KeyOpenAIModel, "gpt-4o-mini")
	viper.SetDefault(KeyOpenAIMaxHistory, 50)
	viper.SetDefault(KeyOpenAISystemPrompt, "You are a helpful assistant.")
	viper.SetDefault(KeyOpenAIMaxToolIterations, 5)
	viper.SetDefault(KeyOpenAIMaxMemoryBytes, 8192)

	// Try to load TOML config file (optional)
	viper.SetConfigName("patrizio")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/patrizio/")

	// Ignore error if config file not found — env vars and defaults still work
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	// Parse allowed chat IDs from config (viper returns []interface{} for lists)
	var allowedChatIDs []int64
	rawIDs := viper.Get(KeyOpenAIAllowedChatIDs)
	if rawIDs != nil {
		switch ids := rawIDs.(type) {
		case []interface{}:
			for _, id := range ids {
				switch v := id.(type) {
				case int64:
					allowedChatIDs = append(allowedChatIDs, v)
				case int:
					allowedChatIDs = append(allowedChatIDs, int64(v))
				case float64:
					allowedChatIDs = append(allowedChatIDs, int64(v))
				}
			}
		case string:
			for _, part := range strings.Split(ids, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				v, err := strconv.ParseInt(part, 10, 64)
				if err == nil {
					allowedChatIDs = append(allowedChatIDs, v)
				}
			}
		}
	}

	return &Config{
		dbPath:                  viper.GetString(KeyDBPath),
		logLevel:                viper.GetString(KeyLogLevel),
		mediaPath:               viper.GetString(KeyMediaPath),
		chatStatePath:           viper.GetString(KeyChatStatePath),
		openAIBaseURL:           viper.GetString(KeyOpenAIBaseURL),
		openAIAPIKey:            viper.GetString(KeyOpenAIAPIKey),
		openAIModel:             viper.GetString(KeyOpenAIModel),
		openAIMaxHistory:        viper.GetInt(KeyOpenAIMaxHistory),
		openAIAllowedChatIDs:    allowedChatIDs,
		openAISystemPrompt:      viper.GetString(KeyOpenAISystemPrompt),
		openAIMaxToolIterations: viper.GetInt(KeyOpenAIMaxToolIterations),
		openAIMaxMemoryBytes:    viper.GetInt(KeyOpenAIMaxMemoryBytes),
	}, nil
}
