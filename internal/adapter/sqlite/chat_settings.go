package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/polpetta/patrizio/internal/database/queries"
)

// ChatSettings implements domain.ChatSettingsRepository using SQLite.
type ChatSettings struct {
	queries *queries.Queries
}

// NewChatSettings creates a ChatSettings backed by db.
func NewChatSettings(db *sql.DB) *ChatSettings {
	return &ChatSettings{queries: queries.New(db)}
}

// Get retrieves a per-chat setting. Returns (value, true, nil) when found, ("", false, nil) when absent.
func (s *ChatSettings) Get(ctx context.Context, chatID int64, key string) (string, bool, error) {
	value, err := s.queries.GetChatSetting(ctx, queries.GetChatSettingParams{ChatID: chatID, Key: key})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get chat setting: %w", err)
	}
	return value, true, nil
}

// Set upserts a per-chat setting value.
func (s *ChatSettings) Set(ctx context.Context, chatID int64, key, value string) error {
	if err := s.queries.UpsertChatSetting(ctx, queries.UpsertChatSettingParams{ChatID: chatID, Key: key, Value: value}); err != nil {
		return fmt.Errorf("failed to set chat setting: %w", err)
	}
	return nil
}

// Delete removes a per-chat setting (no-op if absent).
func (s *ChatSettings) Delete(ctx context.Context, chatID int64, key string) error {
	if err := s.queries.DeleteChatSetting(ctx, queries.DeleteChatSettingParams{ChatID: chatID, Key: key}); err != nil {
		return fmt.Errorf("failed to delete chat setting: %w", err)
	}
	return nil
}
