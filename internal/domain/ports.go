package domain

import "context"

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

// Config defines configuration access
type Config interface {
	DBPath() string
	LogLevel() string
	MediaPath() string
	OpenAIBaseURL() string
	OpenAIAPIKey() string
	OpenAIModel() string
	OpenAIMaxHistory() int
	OpenAIAllowedChatIDs() []int64
	OpenAISystemPrompt() string
}

// AIClient defines the port for AI chat completion services.
type AIClient interface {
	ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error)
}

// ConversationRepository defines database operations for conversation threads.
type ConversationRepository interface {
	SaveMessage(ctx context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string) error
	GetThreadChain(ctx context.Context, leafMsgID int64, limit int) ([]ChatMessage, error)
	IsConversationMessage(ctx context.Context, msgID int64) (bool, *int64, error)
}
