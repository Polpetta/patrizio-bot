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

	if cfg.DBPath() != "./patrizio.db" {
		t.Errorf("DBPath() = %q, want %q", cfg.DBPath(), "./patrizio.db")
	}
	if cfg.LogLevel() != "info" {
		t.Errorf("LogLevel() = %q, want %q", cfg.LogLevel(), "info")
	}
	if cfg.MediaPath() != "./media" {
		t.Errorf("MediaPath() = %q, want %q", cfg.MediaPath(), "./media")
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	os.Clearenv()
	os.Setenv("PATRIZIO_DB_PATH", "/custom/db.db")
	os.Setenv("PATRIZIO_LOG_LEVEL", "debug")
	os.Setenv("PATRIZIO_MEDIA_PATH", "/custom/media")
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
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Change to the temp dir so Load() can find the config
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

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
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}

	// Change to the temp dir so Load() can find the config
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

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
