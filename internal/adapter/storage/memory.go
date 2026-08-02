package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/polpetta/patrizio/internal/domain"
	"github.com/spf13/afero"
)

const memoryFileName = "memory.md"

// MemoryStorage implements domain.MemoryRepository using the filesystem (afero).
// It enforces read-before-write using SHA256 checksums. Callers must Read first
// to obtain a valid token before calling Write, Append, or Clear.
type MemoryStorage struct {
	fs         afero.Fs
	root       string
	maxBytes   int
	settings   domain.ChatSettingsRepository
	readTokens domain.ReadTokensRepository
}

// neverReadError is returned when a write/append/clear is attempted without a prior Read call.
type neverReadError struct {
	msgID int64
}

// Error returns a descriptive message indicating which msgID has never been read.
func (e *neverReadError) Error() string {
	return fmt.Sprintf("memory was never read thread ending where msgID=%d belongs", e.msgID)
}

// NewMemoryStorage creates a MemoryStorage rooted at root with the given size cap, settings backend, and read-token repository.
func NewMemoryStorage(fs afero.Fs, root string, maxBytes int, settings domain.ChatSettingsRepository, readTokens domain.ReadTokensRepository) *MemoryStorage {
	return &MemoryStorage{fs: fs, root: root, maxBytes: maxBytes, settings: settings, readTokens: readTokens}
}

func (m *MemoryStorage) memoryPath(chatID int64) string {
	return filepath.Join(m.root, fmt.Sprintf("%d", chatID), memoryFileName)
}

func (m *MemoryStorage) chatDir(chatID int64) string {
	return filepath.Join(m.root, fmt.Sprintf("%d", chatID))
}

// calculateChecksum returns a hex-encoded SHA-256 of the given content.
// Hex encoding avoids NUL bytes and non-UTF-8 data being stored in a SQLite
// TEXT column, which otherwise truncates/round-trips incorrectly.
func (m *MemoryStorage) calculateChecksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// saveChecksum calculates a SHA256 checksum of the content and persists it via the readTokens repository for later validation.
func (m *MemoryStorage) saveChecksum(ctx context.Context, msgID int64, content []byte) error {
	checksum := m.calculateChecksum(content)
	return m.readTokens.SaveToken(ctx, msgID, checksum)
}

// isChecksumValid checks whether the provided content matches the saved SHA256 checksum for the given message ID.
func (m *MemoryStorage) isChecksumValid(ctx context.Context, msgID int64, provided []byte) (bool, error) {
	token, err := m.readTokens.GetToken(ctx, msgID)
	if err != nil {
		return false, fmt.Errorf("failed to read token: %w", err)
	}
	if token == "" {
		return false, &neverReadError{msgID: msgID}
	}
	return token == m.calculateChecksum(provided), nil
}

// Read returns the memory content for a chat, or "" if no file exists yet. It saves a SHA256 checksum keyed by msgID for later validation in Write/Append/Clear calls.
func (m *MemoryStorage) Read(ctx context.Context, chatID int64, msgID int64) (string, error) {
	data, err := afero.ReadFile(m.fs, m.memoryPath(chatID))
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			if saveErr := m.saveChecksum(ctx, msgID, []byte{}); saveErr != nil {
				return "", fmt.Errorf("failed to save checksum: %w", saveErr)
			}
			return "", nil
		}
		// afero wraps os errors; fall back to an Exists check for other not-found variants.
		ok, existErr := afero.Exists(m.fs, m.memoryPath(chatID))
		if existErr == nil && !ok {
			if saveErr := m.saveChecksum(ctx, msgID, []byte{}); saveErr != nil {
				return "", fmt.Errorf("failed to save checksum: %w", saveErr)
			}
			return "", nil
		}
		return "", fmt.Errorf("failed to read memory: %w", err)
	}

	if err := m.saveChecksum(ctx, msgID, data); err != nil {
		return "", fmt.Errorf("failed to save checksum: %w", err)
	}

	return string(data), nil
}

// Write atomically replaces the memory file via tempfile+rename. Requires readToken from a prior Read call. Returns an error message in the string return if memory was never read or has changed.
func (m *MemoryStorage) Write(ctx context.Context, readToken string, chatID int64, msgID int64, content string) (string, error) {
	if ok, err := m.isChecksumValid(ctx, msgID, []byte(readToken)); err != nil {
		var nre *neverReadError
		if errors.As(err, &nre) {
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

// Append adds text on a new line to the memory file. Requires readToken from a prior Read call. Returns an error message in the string return if memory was never read or has changed.
func (m *MemoryStorage) Append(ctx context.Context, readToken string, chatID int64, msgID int64, text string) (string, error) {
	if ok, err := m.isChecksumValid(ctx, msgID, []byte(readToken)); err != nil {
		var nre *neverReadError
		if errors.As(err, &nre) {
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
	out, err := m.Write(ctx, readToken, chatID, msgID, sb.String())
	if err != nil {
		return "", fmt.Errorf("error: %w", err)
	}
	return out, nil
}

// Clear deletes the memory file for a chat (no-op if absent). Requires readToken from a prior Read call. Returns an error message in the string return if memory was never read or has changed.
func (m *MemoryStorage) Clear(ctx context.Context, readToken string, chatID int64, msgID int64) (string, error) {
	if ok, err := m.isChecksumValid(ctx, msgID, []byte(readToken)); err != nil {
		var nre *neverReadError
		if errors.As(err, &nre) {
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
