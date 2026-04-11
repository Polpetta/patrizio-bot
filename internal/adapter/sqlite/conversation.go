package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/polpetta/patrizio/internal/database/queries"
	"github.com/polpetta/patrizio/internal/domain"
)

// ConversationRepository implements the domain ConversationRepository port using SQLite.
type ConversationRepository struct {
	queries *queries.Queries
}

// NewConversationRepository creates a new ConversationRepository instance.
func NewConversationRepository(db *sql.DB) *ConversationRepository {
	return &ConversationRepository{
		queries: queries.New(db),
	}
}

// SaveMessage persists a conversation message to the database.
func (r *ConversationRepository) SaveMessage(ctx context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string, senderName string) error {
	params := queries.InsertConversationMessageParams{
		ThreadRootID: threadRootID,
		MsgID:        msgID,
		Role:         role,
		Content:      content,
		SenderName:   senderName,
	}

	if parentMsgID != nil {
		params.ParentMsgID = sql.NullInt64{Int64: *parentMsgID, Valid: true}
	}

	if err := r.queries.InsertConversationMessage(ctx, params); err != nil {
		return fmt.Errorf("failed to insert conversation message: %w", err)
	}

	return nil
}

// GetThreadChain retrieves the conversation chain from a leaf message up to the root,
// ordered chronologically, limited to the specified number of messages.
func (r *ConversationRepository) GetThreadChain(ctx context.Context, leafMsgID int64, limit int) ([]domain.ChatMessage, error) {
	rows, err := r.queries.GetThreadChain(ctx, queries.GetThreadChainParams{
		LeafMsgID:   leafMsgID,
		MaxMessages: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get thread chain: %w", err)
	}

	messages := make([]domain.ChatMessage, len(rows))
	for i, row := range rows {
		messages[i] = domain.ChatMessage{
			Role:    row.Role,
			Name:    row.SenderName,
			Content: row.Content,
		}
	}

	return messages, nil
}

// IsConversationMessage checks whether a message ID exists in the conversation store.
// Returns (exists, threadRootID, error). If the message does not exist, exists is false
// and threadRootID is nil.
func (r *ConversationRepository) IsConversationMessage(ctx context.Context, msgID int64) (bool, *int64, error) {
	threadRootID, err := r.queries.IsConversationMessage(ctx, msgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to check conversation message: %w", err)
	}

	return true, &threadRootID, nil
}
