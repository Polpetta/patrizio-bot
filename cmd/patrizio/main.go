// Package main is the entrypoint for the Patrizio Delta Chat bot.
package main

import (
	"os"
	"path/filepath"

	"github.com/polpetta/patrizio/internal/bot"
	"github.com/polpetta/patrizio/internal/config"
	"github.com/polpetta/patrizio/migrations"
)

func main() {
	config.Load()
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Ensure media directory exists
	if err := os.MkdirAll(cfg.MediaPath(), 0755); err != nil {
		panic(err)
	}

	// Ensure database directory exists (SQLite creates the file, but not parent dirs)
	dbDir := filepath.Dir(cfg.DBPath())
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		panic(err)
	}

	// Open database first to build dependencies
	db, err := bot.InitDatabase(cfg, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	// Build dependencies with real adapters
	deps := bot.BuildDependencies(cfg, db)

	// Setup bot with dependencies
	cli := bot.Setup(deps, migrations.FS, ".")

	if err := cli.Start(); err != nil {
		cli.Logger.Error(err)
		os.Exit(1)
	}
}
