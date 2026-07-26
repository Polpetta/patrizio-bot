package sqlite

import (
	"context"
	"testing"
)

func TestChatSettings_GetMissingReturnsFalse(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	s := NewChatSettings(db)
	ctx := context.Background()

	_, ok, err := s.Get(ctx, 42, "memory.enabled")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if ok {
		t.Error("expected ok=false for missing key, got true")
	}
}

func TestChatSettings_SetThenGet(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	s := NewChatSettings(db)
	ctx := context.Background()

	if err := s.Set(ctx, 42, "memory.enabled", "false"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, ok, err := s.Get(ctx, 42, "memory.enabled")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true after Set")
	}
	if value != "false" {
		t.Errorf("value = %q, want %q", value, "false")
	}
}

func TestChatSettings_UpsertOverwrites(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	s := NewChatSettings(db)
	ctx := context.Background()

	_ = s.Set(ctx, 1, "memory.enabled", "false")
	_ = s.Set(ctx, 1, "memory.enabled", "true")

	value, ok, err := s.Get(ctx, 1, "memory.enabled")
	if err != nil || !ok {
		t.Fatalf("Get failed: %v (ok=%v)", err, ok)
	}
	if value != "true" {
		t.Errorf("value = %q, want %q", value, "true")
	}
}

func TestChatSettings_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	s := NewChatSettings(db)
	ctx := context.Background()

	_ = s.Set(ctx, 5, "memory.enabled", "false")
	if err := s.Delete(ctx, 5, "memory.enabled"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok, err := s.Get(ctx, 5, "memory.enabled")
	if err != nil {
		t.Fatalf("Get after Delete failed: %v", err)
	}
	if ok {
		t.Error("expected ok=false after Delete")
	}
}

func TestChatSettings_MultipleKeysPerChat(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	s := NewChatSettings(db)
	ctx := context.Background()

	_ = s.Set(ctx, 10, "memory.enabled", "false")
	_ = s.Set(ctx, 10, "prompt.system_prompt", "be terse")

	v1, ok1, _ := s.Get(ctx, 10, "memory.enabled")
	v2, ok2, _ := s.Get(ctx, 10, "prompt.system_prompt")

	if !ok1 || v1 != "false" {
		t.Errorf("memory.enabled = %q (ok=%v), want %q", v1, ok1, "false")
	}
	if !ok2 || v2 != "be terse" {
		t.Errorf("prompt.system_prompt = %q (ok=%v), want %q", v2, ok2, "be terse")
	}
}

func TestChatSettings_ChatsAreIsolated(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	s := NewChatSettings(db)
	ctx := context.Background()

	_ = s.Set(ctx, 100, "memory.enabled", "false")

	_, ok, err := s.Get(ctx, 200, "memory.enabled")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if ok {
		t.Error("expected ok=false for different chatID")
	}
}
