// Package database provides SQLite connection and migration management.
package database

import (
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	// Pure Go SQLite driver — no CGO required.
	_ "modernc.org/sqlite"
)

// Open opens a SQLite database connection at the given path.
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	return db, nil
}

// Migrate runs all pending forward-only migrations using goose.
// The migrationsFS should contain the migration files at the given dirPath.
func Migrate(db *sql.DB, migrationsFS fs.FS, dirPath string) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, dirPath); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
