package sqlite

import (
	"context"
	"strings"
	"testing"
)

// TestReadTokens_RoundTripPreservesArbitraryBytes is a regression test for a
// bug where checksums were stored as raw SHA-256 bytes cast to a Go string.
// SQLite's TEXT column truncates on the first NUL byte when read back via
// modernc.org/sqlite, causing checksum comparisons to fail intermittently
// (~12% of the time, whenever any SHA byte was 0x00).
//
// The fix is to hex-encode checksums before storing them. This test ensures
// that whatever the caller stores comes back exactly as written — even if it
// contains embedded NULs — so any future regression in encoding is caught.
func TestReadTokens_RoundTripPreservesArbitraryBytes(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Seed a conversation_messages row so GetToken's recursive CTE can find
	// the token by msg_id.
	if _, err := db.Exec(
		`INSERT INTO conversation_messages (thread_root_id, msg_id, parent_msg_id, role, content) VALUES (?, ?, NULL, 'user', 'x')`,
		int64(169), int64(169),
	); err != nil {
		t.Fatalf("failed to seed conversation_messages: %v", err)
	}

	repo := NewReadTokens(db)
	ctx := context.Background()

	cases := []struct {
		name  string
		token string
	}{
		{"plain ascii", "deadbeef"},
		{"embedded nul", "abc\x00def"},
		{"leading nul", "\x00\x00\x00hello"},
		{"all nuls", "\x00\x00\x00\x00"},
		{"non-utf8 bytes", "\xff\xfe\xfd\x00\x01\x02"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := repo.SaveToken(ctx, 169, tc.token); err != nil {
				t.Fatalf("SaveToken failed: %v", err)
			}
			got, err := repo.GetToken(ctx, 169)
			if err != nil {
				t.Fatalf("GetToken failed: %v", err)
			}
			if got != tc.token {
				t.Errorf("round-trip mismatch:\n  saved (%d bytes) = %q\n  got   (%d bytes) = %q",
					len(tc.token), tc.token, len(got), got)
			}
		})
	}
}

// TestReadTokens_GetMissingReturnsEmpty verifies that a msg_id with no saved
// token (and no conversation ancestor with one) yields "" rather than an error.
func TestReadTokens_GetMissingReturnsEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewReadTokens(db)
	ctx := context.Background()

	got, err := repo.GetToken(ctx, 999)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty token for missing msg_id, got %q", got)
	}
}

// TestReadTokens_SaveOverwrites verifies that SaveToken on an existing msg_id
// replaces the previous value (ON CONFLICT DO UPDATE).
func TestReadTokens_SaveOverwrites(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(
		`INSERT INTO conversation_messages (thread_root_id, msg_id, parent_msg_id, role, content) VALUES (?, ?, NULL, 'user', 'x')`,
		int64(1), int64(1),
	); err != nil {
		t.Fatalf("failed to seed conversation_messages: %v", err)
	}

	repo := NewReadTokens(db)
	ctx := context.Background()

	if err := repo.SaveToken(ctx, 1, "first"); err != nil {
		t.Fatalf("SaveToken first failed: %v", err)
	}
	if err := repo.SaveToken(ctx, 1, "second"); err != nil {
		t.Fatalf("SaveToken second failed: %v", err)
	}

	got, err := repo.GetToken(ctx, 1)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if got != "second" {
		t.Errorf("expected overwritten value 'second', got %q", got)
	}
	if strings.Contains(got, "first") {
		t.Errorf("previous value leaked into result: %q", got)
	}
}
