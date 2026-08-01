package sqlite

import (
	"context"

	"github.com/polpetta/patrizio/internal/database/queries"
)

// ReadTokens implements domain.ReadTokensRepository using SQLite.
type ReadTokens struct {
	queries *queries.Queries
}

func NewReadTokens(queries *queries.Queries) *ReadTokens {
	return &ReadTokens{queries: queries}
}

func (r *ReadTokens) GetToken(ctx context.Context, msgID int64) (string, error) {
	observed_sha, err := r.queries.GetReadToken(ctx, msgID)
	if err != nil {
		return "", err
	}
	if observed_sha == "" {
		return "", nil
	}
	return observed_sha, nil
}

func (r *ReadTokens) SaveToken(ctx context.Context, msgID int64, token string) error {
	params := queries.SaveTokenParams{
		MsgID:       msgID,
		ObservedSha: token,
	}
	err := r.queries.SaveToken(ctx, params)
	return err
}
