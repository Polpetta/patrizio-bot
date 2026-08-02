package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/polpetta/patrizio/internal/domain"
)

// fakeMemoryRepo is an in-memory MemoryRepository for testing.
type fakeMemoryRepo struct {
	readContent string // canned content that Read returns
	lastToken   string // records the readToken passed to Append / Write
}

// compile-time check that fakeMemoryRepo implements domain.MemoryRepository.
var _ domain.MemoryRepository = (*fakeMemoryRepo)(nil)

func (f *fakeMemoryRepo) Read(_ context.Context, _ int64, _ int64) (string, error) {
	return f.readContent, nil
}

func (f *fakeMemoryRepo) Append(_ context.Context, readToken string, _ int64, _ int64, _ string) (string, error) {
	f.lastToken = readToken
	if readToken == f.readContent {
		return "", nil
	}
	return "error: call read_memory before modifying memory", nil
}

func (f *fakeMemoryRepo) Write(_ context.Context, readToken string, _ int64, _ int64, _ string) (string, error) {
	f.lastToken = readToken
	if readToken == f.readContent {
		return "", nil
	}
	return "error: call read_memory before modifying memory", nil
}

func (f *fakeMemoryRepo) Clear(_ context.Context, _ string, _ int64, _ int64) (string, error) {
	return "", nil
}

func (f *fakeMemoryRepo) IsEnabled(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

func (f *fakeMemoryRepo) SetEnabled(_ context.Context, _ int64, _ bool) error {
	return nil
}

const (
	testChatID = int64(1)
	testMsgID  = int64(1)
)

var (
	testAppendArgs = json.RawMessage(`{"text":"some note"}`)
	testUpdateArgs = json.RawMessage(`{"content":"fresh content"}`)
)

// newTestHandler builds a handler backed by a fake repo whose Read returns
// "memory-content". The returned *fakeMemoryRepo lets tests inspect lastToken.
func newTestHandler() (*memoryToolHandler, *fakeMemoryRepo) {
	fake := &fakeMemoryRepo{readContent: "memory-content"}
	return NewMemoryToolHandler(fake, testChatID, testMsgID).(*memoryToolHandler), fake
}

func TestMemoryToolHandler_FreshHandlerWroteFalse(t *testing.T) {
	h, _ := newTestHandler()
	if h.Wrote {
		t.Error("expected Wrote=false on fresh handler")
	}
}

func TestMemoryToolHandler_AppendWithoutReadSoftError(t *testing.T) {
	h, fake := newTestHandler()

	result, err := h.Handle(context.Background(), "append_memory", testAppendArgs)
	if err != nil {
		t.Fatalf("expected no error from append with soft-error, got: %v", err)
	}
	if !strings.Contains(result, "call read_memory before modifying memory") {
		t.Errorf("result = %q, want substring %q", result, "call read_memory before modifying memory")
	}
	if fake.lastToken != "" {
		t.Errorf("lastToken = %q, want empty string (no prior read)", fake.lastToken)
	}
	// Wrote is set to true here because the repo returned err == nil (soft error).
	// This is pre-existing behaviour of the handler — Wrote tracks whether append/update
	// was invoked at all (regardless of soft-error), not whether it succeeded.
	if !h.Wrote {
		t.Error("expected Wrote=true after append_memory call (even with soft error)")
	}
}

func TestMemoryToolHandler_ReadReturnsContent(t *testing.T) {
	h, _ := newTestHandler()

	result, err := h.Handle(context.Background(), "read_memory", nil)
	if err != nil {
		t.Fatalf("read_memory failed: %v", err)
	}
	if result != "memory-content" {
		t.Errorf("result = %q, want %q", result, "memory-content")
	}
	if h.Wrote {
		t.Error("expected Wrote=false after read_memory")
	}
}

func TestMemoryToolHandler_ReadThenAppendThreadsToken(t *testing.T) {
	h, fake := newTestHandler()

	// First read to capture the token.
	if _, err := h.Handle(context.Background(), "read_memory", nil); err != nil {
		t.Fatalf("read_memory failed: %v", err)
	}

	// Now append — should use the captured readToken.
	result, err := h.Handle(context.Background(), "append_memory", testAppendArgs)
	if err != nil {
		t.Fatalf("append_memory failed: %v", err)
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
	if fake.lastToken != "memory-content" {
		t.Errorf("lastToken = %q, want %q", fake.lastToken, "memory-content")
	}
	if !h.Wrote {
		t.Error("expected Wrote=true after append_memory")
	}
}

func TestMemoryToolHandler_ReadThenUpdateThreadsToken(t *testing.T) {
	h, fake := newTestHandler()

	// First read to capture the token.
	if _, err := h.Handle(context.Background(), "read_memory", nil); err != nil {
		t.Fatalf("read_memory failed: %v", err)
	}

	// Now update — should use the captured readToken.
	result, err := h.Handle(context.Background(), "update_memory", testUpdateArgs)
	if err != nil {
		t.Fatalf("update_memory failed: %v", err)
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
	if fake.lastToken != "memory-content" {
		t.Errorf("lastToken = %q, want %q", fake.lastToken, "memory-content")
	}
	if !h.Wrote {
		t.Error("expected Wrote=true after update_memory")
	}
}

func TestMemoryToolHandler_UnknownToolReturnsError(t *testing.T) {
	h, _ := newTestHandler()

	result, err := h.Handle(context.Background(), "bogus", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown memory tool") {
		t.Errorf("err.Error() = %q, want substring %q", err.Error(), "unknown memory tool")
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
}

func TestBuildMemoryTools(t *testing.T) {
	tools := BuildMemoryTools()
	if len(tools) != 3 {
		t.Fatalf("BuildMemoryTools() returned %d tools, want 3", len(tools))
	}
	if tools[0].Name != "read_memory" {
		t.Errorf("tools[0].Name = %q, want %q", tools[0].Name, "read_memory")
	}
	if tools[1].Name != "append_memory" {
		t.Errorf("tools[1].Name = %q, want %q", tools[1].Name, "append_memory")
	}
	if tools[2].Name != "update_memory" {
		t.Errorf("tools[2].Name = %q, want %q", tools[2].Name, "update_memory")
	}
}
