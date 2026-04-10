package domain

import "time"

// Response type constants
const (
	ResponseTypeText     = "text"
	ResponseTypeMedia    = "media"
	ResponseTypeReaction = "reaction"
)

// Media type constants
const (
	MediaTypeImage   = "image"
	MediaTypeSticker = "sticker"
	MediaTypeGIF     = "gif"
	MediaTypeVideo   = "video"
)

// Filter represents a stored filter with its metadata
type Filter struct {
	ID           int64
	ChatID       int64
	ResponseType string
	CreatedAt    time.Time
	Triggers     []FilterTrigger
	Response     FilterResponse
}

// FilterTrigger represents a trigger word/phrase for a filter
type FilterTrigger struct {
	ID          int64
	FilterID    int64
	TriggerText string
}

// ChatMessage represents a single message in a conversation thread.
type ChatMessage struct {
	Role    string
	Content string
}

// ChatType represents the type of a Delta Chat chat.
type ChatType string

const (
	// ChatTypeGroup covers group chats, broadcast lists, and mailing lists.
	ChatTypeGroup ChatType = "group"
	// ChatTypeSingle covers one-to-one direct message chats.
	ChatTypeSingle ChatType = "single"
	// ChatTypeUnknown is returned by adapters for chat types not recognised by the domain.
	ChatTypeUnknown ChatType = "unknown"
)

// DownloadState represents the download state of a message's media.
type DownloadState string

const (
	// DownloadDone indicates the media has been fully downloaded.
	DownloadDone DownloadState = "done"
	// DownloadAvailable indicates the media is available to download but not yet fetched.
	DownloadAvailable DownloadState = "available"
)

// IncomingMessage represents an incoming chat message in domain terms.
type IncomingMessage struct {
	ID            uint64
	ChatID        uint64
	FromID        uint64
	Text          string
	File          string        // local path to attached file, empty if none
	MediaType     string        // domain media type constant (image/sticker/gif/video), empty if not media
	DownloadState DownloadState
	Quote         *QuotedMessage // nil if message is not a reply
}

// QuotedMessage represents the message being replied to.
type QuotedMessage struct {
	MessageID uint64
}

// FilterResponse represents the response associated with a filter
type FilterResponse struct {
	FilterID     int64
	ResponseType string
	// Text response fields
	ResponseText string
	// Media response fields
	MediaHash string
	MediaType string
	// Reaction response fields
	Reaction string
}
