package domain

import (
	"context"
	"encoding/json"
)

// FilterRepository defines database operations for filters
type FilterRepository interface {
	CreateTextFilter(ctx context.Context, chatID int64, triggers []string, responseText string) error
	CreateMediaFilter(ctx context.Context, chatID int64, triggers []string, mediaHash string, mediaType string) error
	CreateReactionFilter(ctx context.Context, chatID int64, triggers []string, reaction string) error
	RemoveTrigger(ctx context.Context, chatID int64, triggerText string) (*string, error)
	RemoveAllFilters(ctx context.Context, chatID int64) ([]string, error)
	ListFilters(ctx context.Context, chatID int64) ([]Filter, error)
	FindMatchingFilters(ctx context.Context, chatID int64, normalizedMessage string) ([]FilterResponse, error)
}

// MediaStorage defines filesystem operations for media files
type MediaStorage interface {
	Save(hash string, data []byte) error
	Delete(hash string) error
	Read(hash string) ([]byte, error)
	Path(hash string) string
	Exists(hash string) (bool, error)
}

// MemoryRepository defines per-chat AI memory operations.
// Read returns "" when no file exists. Write/Append return ErrMemoryTooLarge on overflow.
// IsEnabled defaults to true when no setting is stored.
type MemoryRepository interface {
	Read(ctx context.Context, chatID int64) (string, error)
	Write(ctx context.Context, chatID int64, content string) error
	Append(ctx context.Context, chatID int64, text string) error
	Clear(ctx context.Context, chatID int64) error
	IsEnabled(ctx context.Context, chatID int64) (bool, error)
	SetEnabled(ctx context.Context, chatID int64, enabled bool) error
}

// ChatSettingsRepository defines per-chat key/value storage.
// Keys are dot-namespaced by feature (e.g. "memory.enabled").
// Get returns ("", false, nil) when the key has no stored value.
type ChatSettingsRepository interface {
	Get(ctx context.Context, chatID int64, key string) (value string, ok bool, err error)
	Set(ctx context.Context, chatID int64, key, value string) error
	Delete(ctx context.Context, chatID int64, key string) error
}

// Config defines configuration access
type Config interface {
	DBPath() string
	LogLevel() string
	MediaPath() string
	ChatStatePath() string
	OpenAIBaseURL() string
	OpenAIAPIKey() string
	OpenAIModel() string
	OpenAIMaxHistory() int
	OpenAIAllowedChatIDs() []int64
	OpenAISystemPrompt() string
	OpenAIMaxToolIterations() int
	OpenAIMaxMemoryBytes() int
}

// AITool describes a function the AI model may call during a chat completion.
type AITool struct {
	Name        string
	Description string
	Parameters  json.RawMessage // JSON Schema object
}

// AIToolHandler executes a tool call requested by the model.
type AIToolHandler interface {
	Handle(ctx context.Context, name string, args json.RawMessage) (result string, err error)
}

// ChatResponse is the result of a ChatCompletion call.
type ChatResponse struct {
	Content       string
	MemoryWritten bool // true when append_memory or update_memory was called this turn
}

// AIClient defines the port for AI chat completion. When tools is nil, runs a single-shot completion.
type AIClient interface {
	ChatCompletion(ctx context.Context, messages []ChatMessage, tools []AITool, handler AIToolHandler) (ChatResponse, error)
}

// ChatExecutor serialises per-chat operations: for a given chatID only one fn runs at a time;
// different chatIDs run in parallel.
type ChatExecutor interface {
	Run(ctx context.Context, chatID int64, fn func(context.Context) error) error
}

// Messenger defines the port for Delta Chat messaging. All IDs are uint32 (Delta Chat's native type).
type Messenger interface {
	FetchMessage(accountID uint32, msgID uint32) (*IncomingMessage, error)
	FetchChatType(accountID uint32, chatID uint32) (ChatType, error)
	SendTextReply(accountID uint32, chatID uint32, replyTo uint32, text string) (uint32, error)
	// SendMediaReply sends media as a quote-reply; mediaType is a domain media type constant.
	SendMediaReply(accountID uint32, chatID uint32, replyTo uint32, filePath string, mediaType string) (uint32, error)
	SendReaction(accountID uint32, msgID uint32, reaction string) error
	SendTextMessage(accountID uint32, chatID uint32, text string) error
	DownloadMessage(accountID uint32, msgID uint32) error
	// IsSpecialContact reports whether the contact is a system/device contact to ignore (self, device-chat, info bot).
	IsSpecialContact(fromID uint32) bool
	// FetchContactDisplayName falls back through display name → name → email address.
	FetchContactDisplayName(accountID uint32, contactID uint32) (string, error)
}

// ConversationRepository defines database operations for conversation threads.
type ConversationRepository interface {
	SaveMessage(ctx context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string, senderName string) error
	GetThreadChain(ctx context.Context, leafMsgID int64, limit int) ([]ChatMessage, error)
	IsConversationMessage(ctx context.Context, msgID int64) (bool, *int64, error)
}
