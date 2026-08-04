// Package sqlite provides SQLite-backed implementations of domain repository interfaces.
package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/polpetta/patrizio/internal/database/queries"
)

// ReadTokens implements domain.ReadTokensRepository using SQLite for persisting read-token checksums.
type ReadTokens struct {
	queries *queries.Queries
}

// NewReadTokens creates a ReadTokens backed by the given database connection.
func NewReadTokens(db *sql.DB) *ReadTokens {
	return &ReadTokens{queries: queries.New(db)}
}

// GetToken retrieves the saved checksum for a message ID, or returns an empty string if none exists.
func (r *ReadTokens) GetToken(ctx context.Context, msgID int64) (string, error) {
	observedSha, err := r.queries.GetReadToken(ctx, msgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if observedSha == "" {
		return "", nil
	}
	return observedSha, nil
}

// SaveToken persists a checksum for a message ID.
func (r *ReadTokens) SaveToken(ctx context.Context, msgID int64, token string) error {
	params := queries.SaveTokenParams{
		MsgID:       msgID,
		ObservedSha: token,
	}
	err := r.queries.SaveToken(ctx, params)
	return err
}
