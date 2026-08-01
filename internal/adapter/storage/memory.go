package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/polpetta/patrizio/internal/domain"
	"github.com/spf13/afero"

	"crypto/sha256"
)

const memoryFileName = "memory.md"

// MemoryStorage implements domain.MemoryRepository using the filesystem (afero).
type MemoryStorage struct {
	fs         afero.Fs
	root       string
	maxBytes   int
	settings   domain.ChatSettingsRepository
	readTokens domain.ReadTokensRepository
}

type neverReadError struct {
	msgID int64
}

func (e *neverReadError) Error() string {
	return fmt.Sprintf("memory was never read thread ending where msgID=%d belongs", e.msgID)
}

// NewMemoryStorage creates a MemoryStorage rooted at root with the given size cap and settings backend.
func NewMemoryStorage(fs afero.Fs, root string, maxBytes int, settings domain.ChatSettingsRepository, readTokens domain.ReadTokensRepository) *MemoryStorage {
	return &MemoryStorage{fs: fs, root: root, maxBytes: maxBytes, settings: settings, readTokens: readTokens}
}

func (m *MemoryStorage) memoryPath(chatID int64) string {
	return filepath.Join(m.root, fmt.Sprintf("%d", chatID), memoryFileName)
}

func (m *MemoryStorage) chatDir(chatID int64) string {
	return filepath.Join(m.root, fmt.Sprintf("%d", chatID))
}

// calculateChecksum calculates SHA256 checksum of the given content
func (m *MemoryStorage) calculateChecksum(content []byte) string {
	sha := sha256.New()
	sha.Write(content)
	return string(sha.Sum(nil))
}

// saveChecksum calculates SHA256 checksum and saves it in the repo
func (m *MemoryStorage) saveChecksum(ctx context.Context, msgID int64, content []byte) error {
	checksum := m.calculateChecksum(content)
	return m.readTokens.SaveToken(ctx, msgID, string(checksum))
}

func (m *MemoryStorage) isChecksumValid(ctx context.Context, msgID int64, provided []byte) (bool, error) {
	token, err := m.readTokens.GetToken(ctx, msgID)
	if err != nil {
		return false, fmt.Errorf("failed to read token: %w", err)
	}
	if token == "" {
		return false, &neverReadError{}
	}
	return token == m.calculateChecksum(provided), nil
}

// Read returns the memory content for a chat, or "" if no file exists yet.
func (m *MemoryStorage) Read(ctx context.Context, chatID int64, msgID int64) (string, error) {
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

	if err := m.saveChecksum(ctx, msgID, data); err != nil {
		return "", fmt.Errorf("failed to save checksum: %w", err)
	}

	return string(data), nil
}

// Write atomically replaces the memory file via tempfile+rename.
func (m *MemoryStorage) Write(ctx context.Context, readToken string, chatID int64, msgID int64, content string) (string, error) {
	if ok, err := m.isChecksumValid(ctx, msgID, []byte(readToken)); err != nil {
		if errors.Is(err, &neverReadError{}) {
			return "error: call read_memory before modifying memory", nil
		}
		return "", fmt.Errorf("error: %w", err)
	} else if !ok {
		return "error: memory changed since your last read_memory; call read_memory again and retry", nil
	}

	if len(content) > m.maxBytes {
		return "error: memory too large. Compact your memory before adding new content", nil
	}
	if err := m.fs.MkdirAll(m.chatDir(chatID), 0o750); err != nil {
		return "", fmt.Errorf("failed to create chat directory: %w", err)
	}
	tmp := m.memoryPath(chatID) + ".tmp"
	if err := afero.WriteFile(m.fs, tmp, []byte(content), 0o640); err != nil {
		return "", fmt.Errorf("failed to write memory tempfile: %w", err)
	}
	if err := m.fs.Rename(tmp, m.memoryPath(chatID)); err != nil {
		_ = m.fs.Remove(tmp) //nolint:errcheck // best-effort cleanup of failed tempfile
		return "", fmt.Errorf("failed to rename memory tempfile: %w", err)
	}
	return "", nil
}

// Append adds text on a new line to the memory file.
func (m *MemoryStorage) Append(ctx context.Context, readToken string, chatID int64, msgID int64, text string) (string, error) {
	if ok, err := m.isChecksumValid(ctx, msgID, []byte(readToken)); err != nil {
		if errors.Is(err, &neverReadError{}) {
			return "error: call read_memory before modifying memory", nil
		}
		return "", fmt.Errorf("error: %w", err)
	} else if !ok {
		return "error: memory changed since your last read_memory; call read_memory again and retry", nil
	}

	existing, err := m.Read(ctx, chatID, msgID)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if existing != "" {
		sb.WriteString(existing)
		if !strings.HasSuffix(existing, "\n") {
			sb.WriteByte('\n')
		}
	}
	sb.WriteString(text)
	if out, err := m.Write(ctx, readToken, chatID, msgID, sb.String()); err != nil {
		return "", fmt.Errorf("error: %w", err)
	} else {
		return out, nil
	}
}

// Clear deletes the memory file for a chat (no-op if absent).
func (m *MemoryStorage) Clear(ctx context.Context, readToken string, chatID int64, msgID int64) (string, error) {
	if ok, err := m.isChecksumValid(ctx, msgID, []byte(readToken)); err != nil {
		if errors.Is(err, &neverReadError{}) {
			return "error: call read_memory before modifying memory", nil
		}
		return "", fmt.Errorf("error: %w", err)
	} else if !ok {
		return "error: memory changed since your last read_memory; call read_memory again and retry", nil
	}

	path := m.memoryPath(chatID)
	ok, err := afero.Exists(m.fs, path)
	if err != nil {
		return "", fmt.Errorf("failed to check memory file: %w", err)
	}
	if !ok {
		return "", nil
	}
	if err := m.fs.Remove(path); err != nil {
		return "", fmt.Errorf("failed to clear memory: %w", err)
	}
	return "", nil
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
