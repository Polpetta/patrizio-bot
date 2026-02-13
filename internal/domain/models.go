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
