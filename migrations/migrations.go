// Package migrations provides the embedded SQL migration files for database schema management.
package migrations

import "embed"

// FS contains the embedded SQL migration files.
//
//go:embed *.sql
var FS embed.FS
