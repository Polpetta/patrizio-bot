package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/polpetta/patrizio/internal/domain"
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

func newTestMemoryStorage(fs afero.Fs) *MemoryStorage {
	return NewMemoryStorage(fs, "/chat_state", 8192, newFakeSettings())
}

func TestMemoryStorage_WriteReadRoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	if err := ms.Write(ctx, 1, "hello world"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	got, err := ms.Read(ctx, 1)
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

	got, err := ms.Read(ctx, 99)
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

	_ = ms.Write(ctx, 1, "first line")
	if err := ms.Append(ctx, 1, "second line"); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	got, _ := ms.Read(ctx, 1)
	if !strings.Contains(got, "first line") || !strings.Contains(got, "second line") {
		t.Errorf("Append result = %q, want both lines", got)
	}
}

func TestMemoryStorage_AppendToEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	if err := ms.Append(ctx, 1, "only line"); err != nil {
		t.Fatalf("Append to empty failed: %v", err)
	}
	got, _ := ms.Read(ctx, 1)
	if got != "only line" {
		t.Errorf("Read = %q, want %q", got, "only line")
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	_ = ms.Write(ctx, 1, "some content")
	if err := ms.Clear(ctx, 1); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	got, _ := ms.Read(ctx, 1)
	if got != "" {
		t.Errorf("Read after Clear = %q, want empty", got)
	}
}

func TestMemoryStorage_ClearNonExistent(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := newTestMemoryStorage(fs)
	ctx := context.Background()

	if err := ms.Clear(ctx, 999); err != nil {
		t.Errorf("Clear of non-existent should not fail, got: %v", err)
	}
}

func TestMemoryStorage_WriteMaxBytesExceeded(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := NewMemoryStorage(fs, "/chat_state", 10, newFakeSettings())
	ctx := context.Background()

	err := ms.Write(ctx, 1, "this is more than ten bytes")
	if err == nil {
		t.Fatal("expected error for oversized write, got nil")
	}
	if !strings.Contains(err.Error(), "memory content exceeds") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryStorage_AppendMaxBytesExceeded(t *testing.T) {
	fs := afero.NewMemMapFs()
	ms := NewMemoryStorage(fs, "/chat_state", 20, newFakeSettings())
	ctx := context.Background()

	_ = ms.Write(ctx, 1, "twelve bytes!")
	err := ms.Append(ctx, 1, "this pushes it over the limit easily")
	if err == nil {
		t.Fatal("expected error for oversized append, got nil")
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

// Verify MemoryStorage implements the domain.MemoryRepository interface.
var _ domain.MemoryRepository = (*MemoryStorage)(nil)
