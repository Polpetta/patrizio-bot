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

// Messenger defines the port for chat messaging operations.
// All ID parameters use uint64 to match Delta Chat's native types.
type Messenger interface {
	// FetchMessage retrieves a message by ID.
	FetchMessage(accountID uint64, msgID uint64) (*IncomingMessage, error)
	// FetchChatType retrieves the chat type (group/single) for a chat.
	FetchChatType(accountID uint64, chatID uint64) (ChatType, error)
	// SendTextReply sends a text message as a quote-reply. Returns the sent message ID.
	SendTextReply(accountID uint64, chatID uint64, replyTo uint64, text string) (uint64, error)
	// SendMediaReply sends a media file as a quote-reply. mediaType is a domain media type constant.
	SendMediaReply(accountID uint64, chatID uint64, replyTo uint64, filePath string, mediaType string) (uint64, error)
	// SendReaction sends a reaction emoji on a message.
	SendReaction(accountID uint64, msgID uint64, reaction string) error
	// SendTextMessage sends a plain text message (no quote-reply).
	SendTextMessage(accountID uint64, chatID uint64, text string) error
	// DownloadMessage downloads a message's full media content.
	DownloadMessage(accountID uint64, msgID uint64) error
	// IsSpecialContact reports whether the given contact ID is a system/device contact
	// that should be ignored by the bot (e.g. self, device-chat, info bot).
	IsSpecialContact(fromID uint64) bool
	// FetchContactDisplayName retrieves the display name for a contact.
	// Falls back to the contact's name, then email address if display name is empty.
	FetchContactDisplayName(accountID uint64, contactID uint64) (string, error)
}

// ConversationRepository defines database operations for conversation threads.
type ConversationRepository interface {
	SaveMessage(ctx context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string, senderName string) error
	GetThreadChain(ctx context.Context, leafMsgID int64, limit int) ([]ChatMessage, error)
	IsConversationMessage(ctx context.Context, msgID int64) (bool, *int64, error)
}
