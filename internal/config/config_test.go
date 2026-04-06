package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DBPath() != "/data/db/patrizio.db" {
		t.Errorf("DBPath() = %q, want %q", cfg.DBPath(), "/data/db/patrizio.db")
	}
	if cfg.LogLevel() != "info" {
		t.Errorf("LogLevel() = %q, want %q", cfg.LogLevel(), "info")
	}
	if cfg.MediaPath() != "/data/media" {
		t.Errorf("MediaPath() = %q, want %q", cfg.MediaPath(), "/data/media")
	}
	if cfg.OpenAIBaseURL() != "" {
		t.Errorf("OpenAIBaseURL() = %q, want %q", cfg.OpenAIBaseURL(), "")
	}
	if cfg.OpenAIAPIKey() != "" {
		t.Errorf("OpenAIAPIKey() = %q, want %q", cfg.OpenAIAPIKey(), "")
	}
	if cfg.OpenAIModel() != "gpt-4o-mini" {
		t.Errorf("OpenAIModel() = %q, want %q", cfg.OpenAIModel(), "gpt-4o-mini")
	}
	if cfg.OpenAIMaxHistory() != 50 {
		t.Errorf("OpenAIMaxHistory() = %d, want %d", cfg.OpenAIMaxHistory(), 50)
	}
	if len(cfg.OpenAIAllowedChatIDs()) != 0 {
		t.Errorf("OpenAIAllowedChatIDs() = %v, want empty", cfg.OpenAIAllowedChatIDs())
	}
	if cfg.OpenAISystemPrompt() != "You are a helpful assistant." {
		t.Errorf("OpenAISystemPrompt() = %q, want %q", cfg.OpenAISystemPrompt(), "You are a helpful assistant.")
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("PATRIZIO_DB_PATH", "/custom/db.db")
	_ = os.Setenv("PATRIZIO_LOG_LEVEL", "debug")
	_ = os.Setenv("PATRIZIO_MEDIA_PATH", "/custom/media")
	_ = os.Setenv("PATRIZIO_OPENAI_BASE_URL", "https://api.openai.com/v1")
	_ = os.Setenv("PATRIZIO_OPENAI_API_KEY", "sk-test-key")
	_ = os.Setenv("PATRIZIO_OPENAI_MODEL", "gpt-4")
	_ = os.Setenv("PATRIZIO_OPENAI_MAX_HISTORY", "100")
	_ = os.Setenv("PATRIZIO_OPENAI_SYSTEM_PROMPT", "You are Patrizio.")
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DBPath() != "/custom/db.db" {
		t.Errorf("DBPath() = %q, want %q", cfg.DBPath(), "/custom/db.db")
	}
	if cfg.LogLevel() != "debug" {
		t.Errorf("LogLevel() = %q, want %q", cfg.LogLevel(), "debug")
	}
	if cfg.MediaPath() != "/custom/media" {
		t.Errorf("MediaPath() = %q, want %q", cfg.MediaPath(), "/custom/media")
	}
	if cfg.OpenAIBaseURL() != "https://api.openai.com/v1" {
		t.Errorf("OpenAIBaseURL() = %q, want %q", cfg.OpenAIBaseURL(), "https://api.openai.com/v1")
	}
	if cfg.OpenAIAPIKey() != "sk-test-key" {
		t.Errorf("OpenAIAPIKey() = %q, want %q", cfg.OpenAIAPIKey(), "sk-test-key")
	}
	if cfg.OpenAIModel() != "gpt-4" {
		t.Errorf("OpenAIModel() = %q, want %q", cfg.OpenAIModel(), "gpt-4")
	}
	if cfg.OpenAIMaxHistory() != 100 {
		t.Errorf("OpenAIMaxHistory() = %d, want %d", cfg.OpenAIMaxHistory(), 100)
	}
	if cfg.OpenAISystemPrompt() != "You are Patrizio." {
		t.Errorf("OpenAISystemPrompt() = %q, want %q", cfg.OpenAISystemPrompt(), "You are Patrizio.")
	}
}

func TestLoad_TOMLOverride(t *testing.T) {
	os.Clearenv()

	// Create a temporary TOML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "patrizio.toml")
	configContent := `
db_path = "/toml/db.db"
log_level = "warn"
media_path = "/toml/media"
`
	// #nosec G306 - Test file needs standard permissions
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Change to the temp dir so Load() can find the config
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DBPath() != "/toml/db.db" {
		t.Errorf("DBPath() = %q, want %q", cfg.DBPath(), "/toml/db.db")
	}
	if cfg.LogLevel() != "warn" {
		t.Errorf("LogLevel() = %q, want %q", cfg.LogLevel(), "warn")
	}
	if cfg.MediaPath() != "/toml/media" {
		t.Errorf("MediaPath() = %q, want %q", cfg.MediaPath(), "/toml/media")
	}
}

func TestLoad_EnvVarOverridesTOML(t *testing.T) {
	os.Clearenv()

	// Create a temporary TOML config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "patrizio.toml")
	configContent := `
db_path = "/toml/db.db"
log_level = "warn"
media_path = "/toml/media"
`
	// #nosec G306 - Test file needs standard permissions
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Change to the temp dir so Load() can find the config
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(tmpDir)

	// Set env vars that should override TOML
	if err := os.Setenv("PATRIZIO_DB_PATH", "/env/db.db"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("PATRIZIO_MEDIA_PATH", "/env/media"); err != nil {
		t.Fatal(err)
	}
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Env vars should override TOML
	if cfg.DBPath() != "/env/db.db" {
		t.Errorf("DBPath() = %q, want %q (env should override TOML)", cfg.DBPath(), "/env/db.db")
	}
	if cfg.LogLevel() != "warn" {
		t.Errorf("LogLevel() = %q, want %q (TOML value, no env override)", cfg.LogLevel(), "warn")
	}
	if cfg.MediaPath() != "/env/media" {
		t.Errorf("MediaPath() = %q, want %q (env should override TOML)", cfg.MediaPath(), "/env/media")
	}
}
