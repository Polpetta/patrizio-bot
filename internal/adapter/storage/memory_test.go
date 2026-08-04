package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/polpetta/patrizio/internal/domain"
	"github.com/spf13/afero"
)

// fakeSettings is an in-memory ChatSettingsRepository for tests.
type fakeSettings struct {
	data map[int64]map[string]string
}

func newFakeSettings() *fakeSettings {
	return &fakeSettings{data: make(map[int64]map[string]string)}
}

func (f *fakeSettings) Get(_ context.Context, chatID int64, key string) (string, bool, error) {
	if m, ok := f.data[chatID]; ok {
		if v, ok := m[key]; ok {
			return v, true, nil
		}
	}
	return "", false, nil
}

func (f *fakeSettings) Set(_ context.Context, chatID int64, key, value string) error {
	if f.data[chatID] == nil {
		f.data[chatID] = make(map[string]string)
	}
	f.data[chatID][key] = value
	return nil
}

func (f *fakeSettings) Delete(_ context.Context, chatID int64, key string) error {
	if m, ok := f.data[chatID]; ok {
		delete(m, key)
	}
	return nil
}

// fakeReadTokens is an in-memory ReadTokensRepository for tests.
type fakeReadTokens struct {
	tokens map[int64]string
}

func newFakeReadTokens() *fakeReadTokens {
	return &fakeReadTokens{tokens: make(map[int64]string)}
}

func (f *fakeReadTokens) GetToken(_ context.Context, msgID int64) (string, error) {
	return f.tokens[msgID], nil
}

func (f *fakeReadTokens) SaveToken(_ context.Context, msgID int64, token string) error {
	f.tokens[msgID] = token
	return nil
}

func newTestMemoryStorage(fs afero.Fs) *MemoryStorage {
	return NewMemoryStorage(fs, "/chat_state", 8192, newFakeSettings(), newFakeReadTokens())
}

// bootstrapMemory creates an empty memory file on the filesystem and reads it,
// so that a checksum is saved for msgID=1 and the caller gets a valid readToken.
func bootstrapMemory(ctx context.Context, t *testing.T, fs afero.Fs, ms *MemoryStorage) string {
	t.Helper()
	const chatID int64 = 1
	if err := fs.MkdirAll(ms.chatDir(chatID), 0750); err != nil {
		t.Fatalf("bootstrap MkdirAll: %v", err)
	}
	if err := afero.WriteFile(fs, ms.memoryPath(chatID), []byte{}, 0644); err != nil {
		t.Fatalf("bootstrap WriteFile: %v", err)
	}
	readToken, err := ms.Read(ctx, chatID, 1)
	if err != nil {
		t.Fatalf("bootstrap Read: %v", err)
	}
	return readToken
}

func TestMemoryStorage_WriteReadRoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	result, err := ms.Write(ctx, readToken, 1, 1, "hello world")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if result != "" {
		t.Errorf("Write result = %q, want empty", result)
	}
	got, err := ms.Read(ctx, 1, 1)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != "hello world" {
		t.Errorf("Read = %q, want %q", got, "hello world")
	}
}

func TestMemoryStorage_ReadEmptyWhenAbsent(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	got, err := ms.Read(ctx, 99, 1)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != "" {
		t.Errorf("Read = %q, want empty string", got)
	}
}

func TestMemoryStorage_Append(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	ms.Write(ctx, readToken, 1, 1, "first line")
	readToken, _ = ms.Read(ctx, 1, 1)
	result, err := ms.Append(ctx, readToken, 1, 1, "second line")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if result != "" {
		t.Errorf("Append result = %q, want empty", result)
	}
	got, _ := ms.Read(ctx, 1, 1)
	if !strings.Contains(got, "first line") || !strings.Contains(got, "second line") {
		t.Errorf("Append result = %q, want both lines", got)
	}
}

func TestMemoryStorage_AppendToEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	result, err := ms.Append(ctx, readToken, 1, 1, "only line")
	if err != nil {
		t.Fatalf("Append to empty failed: %v", err)
	}
	if result != "" {
		t.Errorf("Append result = %q, want empty", result)
	}
	got, _ := ms.Read(ctx, 1, 1)
	if got != "only line" {
		t.Errorf("Read = %q, want %q", got, "only line")
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	ms.Write(ctx, readToken, 1, 1, "some content")
	readToken, _ = ms.Read(ctx, 1, 1)
	result, err := ms.Clear(ctx, readToken, 1, 1)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if result != "" {
		t.Errorf("Clear result = %q, want empty", result)
	}
	got, _ := ms.Read(ctx, 1, 1)
	if got != "" {
		t.Errorf("Read after Clear = %q, want empty", got)
	}
}

func TestMemoryStorage_ClearNonExistent(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	// First clear removes the bootstrapped file
	ms.Clear(ctx, readToken, 1, 1)
	// Second clear on non-existent file using the same readToken: a missing
	// file is treated as empty by the checksum contract, so this still matches
	// the bootstrapped empty-file token and must succeed.
	result, err := ms.Clear(ctx, readToken, 1, 1)
	if err != nil {
		t.Errorf("Clear of non-existent should not fail, got: %v", err)
	}
	if result != "" {
		t.Errorf("Clear result = %q, want empty", result)
	}
}

func TestMemoryStorage_WriteMaxBytesExceeded(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := NewMemoryStorage(fs, "/chat_state", 10, newFakeSettings(), newFakeReadTokens())
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	result, err := ms.Write(ctx, readToken, 1, 1, "this is more than ten bytes")
	if err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}
	if !strings.Contains(result, "memory too large") {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestMemoryStorage_AppendMaxBytesExceeded(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := NewMemoryStorage(fs, "/chat_state", 20, newFakeSettings(), newFakeReadTokens())
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	ms.Write(ctx, readToken, 1, 1, "twelve bytes!")
	readToken, _ = ms.Read(ctx, 1, 1)
	result, err := ms.Append(ctx, readToken, 1, 1, "this pushes it over the limit easily")
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if !strings.Contains(result, "memory too large") {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestMemoryStorage_IsEnabledDefaultTrue(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	enabled, err := ms.IsEnabled(ctx, 1)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("expected IsEnabled=true by default")
	}
}

func TestMemoryStorage_SetEnabledDisable(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	if err := ms.SetEnabled(ctx, 1, false); err != nil {
		t.Fatalf("SetEnabled false failed: %v", err)
	}
	enabled, err := ms.IsEnabled(ctx, 1)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if enabled {
		t.Error("expected IsEnabled=false after SetEnabled(false)")
	}
}

func TestMemoryStorage_SetEnabledReenable(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	_ = ms.SetEnabled(ctx, 1, false)
	_ = ms.SetEnabled(ctx, 1, true)
	enabled, _ := ms.IsEnabled(ctx, 1)
	if !enabled {
		t.Error("expected IsEnabled=true after re-enabling")
	}
}

// --- Read-before-write contract tests ---

func TestMemoryStorage_ReadThenWrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	result, err := ms.Write(ctx, readToken, 1, 1, "hello world")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if result != "" {
		t.Errorf("Write result = %q, want empty", result)
	}
	got, err := ms.Read(ctx, 1, 1)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != "hello world" {
		t.Errorf("Read = %q, want %q", got, "hello world")
	}
}

func TestMemoryStorage_ReadThenAppend(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	ms.Write(ctx, readToken, 1, 1, "base")
	readToken, _ = ms.Read(ctx, 1, 1)
	result, err := ms.Append(ctx, readToken, 1, 1, "appended")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if result != "" {
		t.Errorf("Append result = %q, want empty", result)
	}
	got, _ := ms.Read(ctx, 1, 1)
	if !strings.Contains(got, "base") || !strings.Contains(got, "appended") {
		t.Errorf("Read = %q, want both base and appended", got)
	}
}

func TestMemoryStorage_ReadThenClear(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	ms.Write(ctx, readToken, 1, 1, "content")
	readToken, _ = ms.Read(ctx, 1, 1)
	result, err := ms.Clear(ctx, readToken, 1, 1)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if result != "" {
		t.Errorf("Clear result = %q, want empty", result)
	}
	got, _ := ms.Read(ctx, 1, 1)
	if got != "" {
		t.Errorf("Read after Clear = %q, want empty", got)
	}
}

func TestMemoryStorage_WriteFailsWithoutRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	// No bootstrap, no Read — Write must fail
	result, err := ms.Write(ctx, "", 1, 1, "content")
	if err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}
	if !strings.Contains(result, "call read_memory before modifying memory") {
		t.Errorf("Write result = %q, want error about read_memory", result)
	}
}

func TestMemoryStorage_AppendFailsWithoutRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	// No bootstrap, no Read — Append must fail
	result, err := ms.Append(ctx, "", 1, 1, "text")
	if err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}
	if !strings.Contains(result, "call read_memory before modifying memory") {
		t.Errorf("Append result = %q, want error about read_memory", result)
	}
}

func TestMemoryStorage_ClearFailsWithoutRead(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	// No bootstrap, no Read — Clear must fail
	result, err := ms.Clear(ctx, "", 1, 1)
	if err != nil {
		t.Fatalf("Clear returned unexpected error: %v", err)
	}
	if !strings.Contains(result, "call read_memory before modifying memory") {
		t.Errorf("Clear result = %q, want error about read_memory", result)
	}
}

func TestMemoryStorage_WriteFailsWithStaleReadToken(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	readToken := bootstrapMemory(ctx, t, fs, ms)
	ms.Write(ctx, readToken, 1, 1, "original content")
	// Read again — save this token as the "old" one
	oldToken, _ := ms.Read(ctx, 1, 1)
	// Externally modify the file behind our back
	if err := afero.WriteFile(fs, ms.memoryPath(1), []byte("hacked content"), 0644); err != nil {
		t.Fatalf("failed to externally modify file: %v", err)
	}
	// Re-read so the stored checksum is updated (making oldToken stale)
	ms.Read(ctx, 1, 1)
	// Try to write with the stale token
	result, err := ms.Write(ctx, oldToken, 1, 1, "new content")
	if err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}
	if !strings.Contains(result, "memory changed since your last read_memory") {
		t.Errorf("Write result = %q, want stale token error", result)
	}
}

func TestMemoryStorage_EmptyMemoryReadThenWrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	// Bootstrap creates an empty file; Read returns "" as the readToken
	readToken := bootstrapMemory(ctx, t, fs, ms)
	if readToken != "" {
		t.Fatalf("expected empty readToken for empty memory, got %q", readToken)
	}
	// Write using the empty readToken
	result, err := ms.Write(ctx, readToken, 1, 1, "some content")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if result != "" {
		t.Errorf("Write result = %q, want empty", result)
	}
	// Read back and verify
	got, err := ms.Read(ctx, 1, 1)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got != "some content" {
		t.Errorf("Read = %q, want %q", got, "some content")
	}
}

// Verify MemoryStorage implements the domain.MemoryRepository interface.
var _ domain.MemoryRepository = (*MemoryStorage)(nil)

// TestMemoryStorage_ReadThenWriteOnMissingFile guards against the first-turn
// /prompt regression: when no memory file exists yet, Read stores the checksum
// of empty bytes, so a subsequent Write must not be rejected as stale by the
// concurrency check.
func TestMemoryStorage_ReadThenWriteOnMissingFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	// No bootstrap: memory file does not exist on disk.
	readToken, err := ms.Read(ctx, 1, 1)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if readToken != "" {
		t.Fatalf("expected empty readToken for missing memory, got %q", readToken)
	}
	result, err := ms.Write(ctx, readToken, 1, 1, "hello world")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if result != "" {
		t.Errorf("Write result = %q, want empty", result)
	}
	got, _ := ms.Read(ctx, 1, 1)
	if got != "hello world" {
		t.Errorf("Read = %q, want %q", got, "hello world")
	}
}
