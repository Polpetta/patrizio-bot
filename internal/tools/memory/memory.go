package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/polpetta/patrizio/internal/domain"
)

// ErrMemoryTooLarge is returned when a write would exceed the configured memory size limit.
var ErrMemoryTooLarge = errors.New("memory content exceeds size limit")

const (
	toolReadMemory   = "read_memory"
	toolAppendMemory = "append_memory"
	toolUpdateMemory = "update_memory"
)

// BuildMemoryTools returns the tool descriptors for read_memory, append_memory, and update_memory.
func BuildMemoryTools() []domain.AITool {
	return []domain.AITool{
		{
			Name:        toolReadMemory,
			Description: "Read the current memory file for this chat. Returns the full markdown content, or the string \"(memory is empty)\" if no memory has been written yet.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		},
		{
			Name:        toolAppendMemory,
			Description: "Append a short fact or note to memory. Use for incremental additions. Content is added on a new line.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"],"additionalProperties":false}`),
		},
		{
			Name:        toolUpdateMemory,
			Description: "Replace the entire memory file with new content. Use to reorganise, dedupe, or remove outdated facts.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"content":{"type":"string"}},"required":["content"],"additionalProperties":false}`),
		},
	}
}

// memoryToolHandler implements AIToolHandler for the memory tools.
// Wrote is set to true when append_memory or update_memory is called.
type memoryToolHandler struct {
	repo   domain.MemoryRepository
	chatID int64
	Wrote  bool
}

// NewMemoryToolHandler creates a handler bound to a chat's memory repository.
func NewMemoryToolHandler(repo domain.MemoryRepository, chatID int64) domain.AIToolHandler {
	return &memoryToolHandler{repo: repo, chatID: chatID}
}

// Handle dispatches a named tool call to the appropriate MemoryRepository method.
func (h *memoryToolHandler) Handle(ctx context.Context, name string, args json.RawMessage) (string, error) {
	switch name {
	case toolReadMemory:
		content, err := h.repo.Read(ctx, h.chatID)
		if err != nil {
			return "", fmt.Errorf("read_memory failed: %w", err)
		}
		if content == "" {
			return "(memory is empty)", nil
		}
		return content, nil

	case toolAppendMemory:
		var p struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("append_memory: invalid args: %w", err)
		}
		if err := h.repo.Append(ctx, h.chatID, p.Text); err != nil {
			return "", fmt.Errorf("append_memory failed: %w", err)
		}
		h.Wrote = true
		return "appended", nil

	case toolUpdateMemory:
		var p struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("update_memory: invalid args: %w", err)
		}
		if err := h.repo.Write(ctx, h.chatID, p.Content); err != nil {
			return "", fmt.Errorf("update_memory failed: %w", err)
		}
		h.Wrote = true
		return "updated", nil

	default:
		return "", fmt.Errorf("unknown memory tool: %s", name)
	}
}
