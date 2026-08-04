package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/polpetta/patrizio/internal/adapter/sqlite"
	"github.com/polpetta/patrizio/internal/database"
	"github.com/polpetta/patrizio/migrations"
	"github.com/spf13/afero"
)

// TestMemoryStorage_ReadThenWrite_NoConversationRow reproduces a runtime bug
// observed on /prompt's first turn: handlePromptCommand only inserts into
// conversation_messages AFTER ChatCompletion returns, but the memory tools
// (read_memory / update_memory) run DURING ChatCompletion. The GetReadToken
// recursive CTE seeds from conversation_messages, so on the first turn it
// finds no row for the current msg_id and returns "" — causing every write
// to fail with "call read_memory before modifying memory" even though
// read_memory was just called.
//
// This test wires the real SQLite ReadTokens repository (not the fake) to
// exercise the query as it runs in production.
func TestMemoryStorage_ReadThenWrite_NoConversationRow(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(db, migrations.FS, "."); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fs := afero.NewMemMapFs()
	ms := NewMemoryStorage(fs, "/chat_state", 8192, sqlite.NewChatSettings(db), sqlite.NewReadTokens(db))
	ctx := context.Background()

	const chatID, msgID = int64(1), int64(169)

	readToken, err := ms.Read(ctx, chatID, msgID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	result, err := ms.Write(ctx, readToken, chatID, msgID, "hello world")
	if err != nil {
		t.Fatalf("Write returned unexpected error: %v", err)
	}
	if strings.Contains(result, "call read_memory before modifying memory") {
		t.Fatalf("Write reported neverReadError even though Read was just called: %q", result)
	}
	if strings.Contains(result, "memory changed since your last read_memory") {
		t.Fatalf("Write reported stale-read even though Read was just called: %q", result)
	}
}
