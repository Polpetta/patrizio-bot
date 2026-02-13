// Package sqlite provides the SQLite implementation of the FilterRepository port.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/polpetta/patrizio/internal/database/queries"
	"github.com/polpetta/patrizio/internal/domain"
)

// Repository implements the domain FilterRepository port using SQLite.
type Repository struct {
	db      *sql.DB
	queries *queries.Queries
}

// New creates a new Repository instance.
func New(db *sql.DB) *Repository {
	return &Repository{
		db:      db,
		queries: queries.New(db),
	}
}

// CreateTextFilter creates a text filter with the given triggers and response.
func (r *Repository) CreateTextFilter(ctx context.Context, chatID int64, triggers []string, responseText string) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	// Check for duplicate triggers
	for _, trigger := range triggers {
		normalized := domain.NormalizeTrigger(trigger)
		if err := domain.ValidateTrigger(normalized); err != nil {
			return err
		}

		hasDup, err := qtx.CheckDuplicateTriggerInChat(ctx, queries.CheckDuplicateTriggerInChatParams{
			ChatID:      chatID,
			TriggerText: normalized,
		})
		if err != nil {
			return fmt.Errorf("failed to check for duplicate trigger: %w", err)
		}
		if hasDup != 0 {
			return fmt.Errorf("trigger %q already exists in this chat", trigger)
		}
	}

	// Insert filter
	filterID, err := qtx.InsertFilter(ctx, queries.InsertFilterParams{
		ChatID:       chatID,
		ResponseType: domain.ResponseTypeText,
	})
	if err != nil {
		return fmt.Errorf("failed to insert filter: %w", err)
	}

	// Insert triggers
	for _, trigger := range triggers {
		normalized := domain.NormalizeTrigger(trigger)
		if err := qtx.InsertFilterTrigger(ctx, queries.InsertFilterTriggerParams{
			FilterID:    filterID,
			TriggerText: normalized,
		}); err != nil {
			return fmt.Errorf("failed to insert trigger: %w", err)
		}
	}

	// Insert text response
	if err := qtx.InsertTextResponse(ctx, queries.InsertTextResponseParams{
		FilterID:     filterID,
		ResponseText: responseText,
	}); err != nil {
		return fmt.Errorf("failed to insert text response: %w", err)
	}

	return tx.Commit()
}

// CreateMediaFilter creates a media filter with the given triggers and media reference.
func (r *Repository) CreateMediaFilter(ctx context.Context, chatID int64, triggers []string, mediaHash string, mediaType string) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	// Check for duplicate triggers
	for _, trigger := range triggers {
		normalized := domain.NormalizeTrigger(trigger)
		if err := domain.ValidateTrigger(normalized); err != nil {
			return err
		}

		hasDup, err := qtx.CheckDuplicateTriggerInChat(ctx, queries.CheckDuplicateTriggerInChatParams{
			ChatID:      chatID,
			TriggerText: normalized,
		})
		if err != nil {
			return fmt.Errorf("failed to check for duplicate trigger: %w", err)
		}
		if hasDup != 0 {
			return fmt.Errorf("trigger %q already exists in this chat", trigger)
		}
	}

	// Insert filter
	filterID, err := qtx.InsertFilter(ctx, queries.InsertFilterParams{
		ChatID:       chatID,
		ResponseType: domain.ResponseTypeMedia,
	})
	if err != nil {
		return fmt.Errorf("failed to insert filter: %w", err)
	}

	// Insert triggers
	for _, trigger := range triggers {
		normalized := domain.NormalizeTrigger(trigger)
		if err := qtx.InsertFilterTrigger(ctx, queries.InsertFilterTriggerParams{
			FilterID:    filterID,
			TriggerText: normalized,
		}); err != nil {
			return fmt.Errorf("failed to insert trigger: %w", err)
		}
	}

	// Insert media response
	if err := qtx.InsertMediaResponse(ctx, queries.InsertMediaResponseParams{
		FilterID:  filterID,
		MediaHash: mediaHash,
		MediaType: mediaType,
	}); err != nil {
		return fmt.Errorf("failed to insert media response: %w", err)
	}

	return tx.Commit()
}

// CreateReactionFilter creates a reaction filter with the given triggers and emoji.
func (r *Repository) CreateReactionFilter(ctx context.Context, chatID int64, triggers []string, reaction string) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	// Check for duplicate triggers
	for _, trigger := range triggers {
		normalized := domain.NormalizeTrigger(trigger)
		if err := domain.ValidateTrigger(normalized); err != nil {
			return err
		}

		hasDup, err := qtx.CheckDuplicateTriggerInChat(ctx, queries.CheckDuplicateTriggerInChatParams{
			ChatID:      chatID,
			TriggerText: normalized,
		})
		if err != nil {
			return fmt.Errorf("failed to check for duplicate trigger: %w", err)
		}
		if hasDup != 0 {
			return fmt.Errorf("trigger %q already exists in this chat", trigger)
		}
	}

	// Insert filter
	filterID, err := qtx.InsertFilter(ctx, queries.InsertFilterParams{
		ChatID:       chatID,
		ResponseType: domain.ResponseTypeReaction,
	})
	if err != nil {
		return fmt.Errorf("failed to insert filter: %w", err)
	}

	// Insert triggers
	for _, trigger := range triggers {
		normalized := domain.NormalizeTrigger(trigger)
		if err := qtx.InsertFilterTrigger(ctx, queries.InsertFilterTriggerParams{
			FilterID:    filterID,
			TriggerText: normalized,
		}); err != nil {
			return fmt.Errorf("failed to insert trigger: %w", err)
		}
	}

	// Insert reaction response
	if err := qtx.InsertReactionResponse(ctx, queries.InsertReactionResponseParams{
		FilterID: filterID,
		Reaction: reaction,
	}); err != nil {
		return fmt.Errorf("failed to insert reaction response: %w", err)
	}

	return tx.Commit()
}

// RemoveTrigger removes a trigger from a chat. Returns the media hash if the filter was deleted and was a media filter.
func (r *Repository) RemoveTrigger(ctx context.Context, chatID int64, triggerText string) (*string, error) {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	// Normalize the trigger text
	normalized := domain.NormalizeTrigger(triggerText)

	// Delete the trigger and get the filter ID
	filterID, err := qtx.DeleteTriggerByChatAndText(ctx, queries.DeleteTriggerByChatAndTextParams{
		ChatID:      chatID,
		TriggerText: normalized,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("trigger %q not found in this chat", triggerText)
		}
		return nil, fmt.Errorf("failed to delete trigger: %w", err)
	}

	// Check if the filter has any remaining triggers
	count, err := qtx.CountTriggersByFilterID(ctx, filterID)
	if err != nil {
		return nil, fmt.Errorf("failed to count remaining triggers: %w", err)
	}

	var mediaHash *string

	// If no triggers remain, delete the filter and check if we need to clean up media
	if count == 0 {
		// Get filter info to check if it's a media filter
		filter, err := qtx.GetFilterByID(ctx, filterID)
		if err != nil {
			return nil, fmt.Errorf("failed to get filter info: %w", err)
		}

		// If it's a media filter, get the hash before deletion
		if filter.ResponseType == domain.ResponseTypeMedia {
			mediaResp, err := qtx.GetMediaResponseByFilterID(ctx, filterID)
			if err != nil {
				return nil, fmt.Errorf("failed to get media response: %w", err)
			}

			// Check if any other filters reference this hash
			refCount, err := qtx.CountMediaResponsesByHash(ctx, mediaResp.MediaHash)
			if err != nil {
				return nil, fmt.Errorf("failed to count media references: %w", err)
			}

			// If this is the last reference, we'll need to delete the file
			if refCount == 1 {
				mediaHash = &mediaResp.MediaHash
			}
		}

		// Delete the filter (cascade will delete the response)
		if err := qtx.DeleteFilter(ctx, filterID); err != nil {
			return nil, fmt.Errorf("failed to delete filter: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return mediaHash, nil
}

// RemoveAllFilters removes all filters from a chat. Returns media hashes that were deleted.
func (r *Repository) RemoveAllFilters(ctx context.Context, chatID int64) ([]string, error) {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	// Get all media hashes for this chat before deletion
	allMediaHashes, err := qtx.GetMediaHashesByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media hashes: %w", err)
	}

	// Get all filters for this chat
	filters, err := qtx.ListFiltersByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list filters: %w", err)
	}

	// Delete all filters (cascade will delete triggers and responses)
	for _, filter := range filters {
		if err := qtx.DeleteFilter(ctx, filter.ID); err != nil {
			return nil, fmt.Errorf("failed to delete filter %d: %w", filter.ID, err)
		}
	}

	// For each media hash, check if it's still referenced by other chats
	var hashesToDelete []string
	for _, hash := range allMediaHashes {
		refCount, err := qtx.CountMediaResponsesByHash(ctx, hash)
		if err != nil {
			return nil, fmt.Errorf("failed to count references for hash %s: %w", hash, err)
		}
		// If no references remain, add to cleanup list
		if refCount == 0 {
			hashesToDelete = append(hashesToDelete, hash)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return hashesToDelete, nil
}

// ListFilters returns all filters for a chat.
func (r *Repository) ListFilters(ctx context.Context, chatID int64) ([]domain.Filter, error) {
	filters, err := r.queries.ListFiltersByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to list filters: %w", err)
	}

	var result []domain.Filter
	for _, f := range filters {
		// Get triggers for this filter
		triggers, err := r.queries.GetTriggersByFilterID(ctx, f.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get triggers for filter %d: %w", f.ID, err)
		}

		domainTriggers := make([]domain.FilterTrigger, len(triggers))
		for i, t := range triggers {
			domainTriggers[i] = domain.FilterTrigger{
				ID:          t.ID,
				FilterID:    t.FilterID,
				TriggerText: t.TriggerText,
			}
		}

		// Build response based on type
		var response domain.FilterResponse
		response.FilterID = f.ID
		response.ResponseType = f.ResponseType

		switch f.ResponseType {
		case domain.ResponseTypeText:
			textResp, err := r.queries.GetTextResponseByFilterID(ctx, f.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get text response for filter %d: %w", f.ID, err)
			}
			response.ResponseText = textResp.ResponseText

		case domain.ResponseTypeMedia:
			mediaResp, err := r.queries.GetMediaResponseByFilterID(ctx, f.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get media response for filter %d: %w", f.ID, err)
			}
			response.MediaHash = mediaResp.MediaHash
			response.MediaType = mediaResp.MediaType

		case domain.ResponseTypeReaction:
			reactionResp, err := r.queries.GetReactionResponseByFilterID(ctx, f.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get reaction response for filter %d: %w", f.ID, err)
			}
			response.Reaction = reactionResp.Reaction
		}

		result = append(result, domain.Filter{
			ID:           f.ID,
			ChatID:       f.ChatID,
			ResponseType: f.ResponseType,
			CreatedAt:    f.CreatedAt,
			Triggers:     domainTriggers,
			Response:     response,
		})
	}

	return result, nil
}

// FindMatchingFilters finds all filters that match the normalized message.
func (r *Repository) FindMatchingFilters(ctx context.Context, chatID int64, normalizedMessage string) ([]domain.FilterResponse, error) {
	rows, err := r.queries.FindMatchingFilters(ctx, queries.FindMatchingFiltersParams{
		ChatID:  chatID,
		Column2: sql.NullString{String: normalizedMessage, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find matching filters: %w", err)
	}

	var result []domain.FilterResponse
	for _, row := range rows {
		response := domain.FilterResponse{
			FilterID:     row.ID,
			ResponseType: row.ResponseType,
		}

		switch row.ResponseType {
		case domain.ResponseTypeText:
			response.ResponseText = row.ResponseText

		case domain.ResponseTypeMedia:
			response.MediaHash = row.MediaHash
			response.MediaType = row.MediaType

		case domain.ResponseTypeReaction:
			response.Reaction = row.Reaction
		}

		result = append(result, response)
	}

	return result, nil
}
