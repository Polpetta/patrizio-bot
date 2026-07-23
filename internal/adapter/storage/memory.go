package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/polpetta/patrizio/internal/domain"
)

const memoryFileName = "memory.md"

// MemoryStorage implements domain.MemoryRepository using the filesystem (afero).
type MemoryStorage struct {
	fs       afero.Fs
	root     string
	maxBytes int
	settings domain.ChatSettingsRepository
}

// NewMemoryStorage creates a MemoryStorage rooted at root with the given size cap and settings backend.
func NewMemoryStorage(fs afero.Fs, root string, maxBytes int, settings domain.ChatSettingsRepository) *MemoryStorage {
	return &MemoryStorage{fs: fs, root: root, maxBytes: maxBytes, settings: settings}
}

func (m *MemoryStorage) memoryPath(chatID int64) string {
	return filepath.Join(m.root, fmt.Sprintf("%d", chatID), memoryFileName)
}

func (m *MemoryStorage) chatDir(chatID int64) string {
	return filepath.Join(m.root, fmt.Sprintf("%d", chatID))
}

// Read returns the memory content for a chat, or "" if no file exists yet.
func (m *MemoryStorage) Read(_ context.Context, chatID int64) (string, error) {
	data, err := afero.ReadFile(m.fs, m.memoryPath(chatID))
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return "", nil
		}
		// afero wraps os errors; fall back to an Exists check for other not-found variants.
		ok, existErr := afero.Exists(m.fs, m.memoryPath(chatID))
		if existErr == nil && !ok {
			return "", nil
		}
		return "", fmt.Errorf("failed to read memory: %w", err)
	}
	return string(data), nil
}

// Write atomically replaces the memory file via tempfile+rename.
func (m *MemoryStorage) Write(_ context.Context, chatID int64, content string) error {
	if len(content) > m.maxBytes {
		return fmt.Errorf("%w: %d bytes (max %d)", domain.ErrMemoryTooLarge, len(content), m.maxBytes)
	}
	if err := m.fs.MkdirAll(m.chatDir(chatID), 0o750); err != nil {
		return fmt.Errorf("failed to create chat directory: %w", err)
	}
	tmp := m.memoryPath(chatID) + ".tmp"
	if err := afero.WriteFile(m.fs, tmp, []byte(content), 0o640); err != nil {
		return fmt.Errorf("failed to write memory tempfile: %w", err)
	}
	if err := m.fs.Rename(tmp, m.memoryPath(chatID)); err != nil {
		_ = m.fs.Remove(tmp) //nolint:errcheck // best-effort cleanup of failed tempfile
		return fmt.Errorf("failed to rename memory tempfile: %w", err)
	}
	return nil
}

// Append adds text on a new line to the memory file.
func (m *MemoryStorage) Append(ctx context.Context, chatID int64, text string) error {
	existing, err := m.Read(ctx, chatID)
	if err != nil {
		return err
	}
	var sb strings.Builder
	if existing != "" {
		sb.WriteString(existing)
		if !strings.HasSuffix(existing, "\n") {
			sb.WriteByte('\n')
		}
	}
	sb.WriteString(text)
	return m.Write(ctx, chatID, sb.String())
}

// Clear deletes the memory file for a chat (no-op if absent).
func (m *MemoryStorage) Clear(_ context.Context, chatID int64) error {
	path := m.memoryPath(chatID)
	ok, err := afero.Exists(m.fs, path)
	if err != nil {
		return fmt.Errorf("failed to check memory file: %w", err)
	}
	if !ok {
		return nil
	}
	if err := m.fs.Remove(path); err != nil {
		return fmt.Errorf("failed to clear memory: %w", err)
	}
	return nil
}

// IsEnabled reports whether memory is enabled for the chat; defaults to true when no setting is stored.
func (m *MemoryStorage) IsEnabled(ctx context.Context, chatID int64) (bool, error) {
	value, ok, err := m.settings.Get(ctx, chatID, "memory.enabled")
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	return value == "true", nil
}

// SetEnabled persists the memory enabled/disabled flag for the chat.
func (m *MemoryStorage) SetEnabled(ctx context.Context, chatID int64, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return m.settings.Set(ctx, chatID, "memory.enabled", value)
}
