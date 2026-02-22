package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"

	"github.com/polpetta/patrizio/internal/domain"
)

const helpText = `Hi! I'm Patrizio, a group chat bot.

Add me to a group and I'll respond to messages based on configured filters.

I don't do much in direct messages — add me to a group to get started!`

var errChatIDOverflow = errors.New("chat ID too large to convert")

// convertChatID safely converts uint64 chat ID to int64 for database operations.
func convertChatID(chatID deltachat.ChatId) (int64, error) {
	// Delta Chat uses uint64 for ChatId, but SQLite's INTEGER PRIMARY KEY is int64.
	// This conversion is safe because chat IDs in practice never exceed MaxInt64.
	if uint64(chatID) > math.MaxInt64 {
		return 0, errChatIDOverflow
	}
	//nolint:gosec // G115: Overflow checked explicitly above
	return int64(chatID), nil
}

// newMsgHandler returns the OnNewMsg callback that routes incoming messages.
func newMsgHandler(cli *botcli.BotCli, _ *deltachat.Bot, deps *domain.Dependencies) deltachat.NewMsgHandler {
	return func(bot *deltachat.Bot, accID deltachat.AccountId, msgID deltachat.MsgId) {
		logger := cli.GetLogger(accID)

		// Extract bot.Rpc as the rpcClient interface so all downstream
		// handler functions are decoupled from *deltachat.Bot and can be
		// tested with a mock.
		rpc := bot.Rpc

		msg, err := rpc.GetMessage(accID, msgID)
		if err != nil {
			logger.Errorf("Failed to get message %d: %v", msgID, err)
			return
		}

		// Ignore messages from special contacts (system, device, etc.).
		if msg.FromId <= deltachat.ContactLastSpecial {
			return
		}

		chatInfo, err := rpc.GetBasicChatInfo(accID, msg.ChatId)
		if err != nil {
			logger.Errorf("Failed to get chat info for chat %d: %v", msg.ChatId, err)
			return
		}

		switch chatInfo.ChatType {
		case deltachat.ChatGroup, deltachat.ChatOutBroadcast, deltachat.ChatInBroadcast, deltachat.ChatMailinglist:
			handleGroupMessage(rpc, logger, accID, msgID, msg, deps)
		case deltachat.ChatSingle:
			handleDMMessage(rpc, logger, accID, msg)
		default:
			logger.Warnf("Unknown chat type %s for chat %d, ignoring", chatInfo.ChatType, msg.ChatId)
		}
	}
}

// handleGroupMessage processes a message from a group chat.
// It checks for commands first, then normalizes the message and checks for matching filters.
func handleGroupMessage(
	rpc rpcClient,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Check if this is a command
	cmdType := domain.GetCommandType(msg.Text)
	if cmdType != "" {
		switch cmdType {
		case domain.CommandFilter:
			handleFilterCommand(rpc, logger, accID, msg, deps)
			return
		case domain.CommandStop:
			handleStopCommand(rpc, logger, accID, msg, deps)
			return
		case domain.CommandStopAll:
			handleStopAllCommand(rpc, logger, accID, msg, deps)
			return
		case domain.CommandFilters:
			handleFiltersCommand(rpc, logger, accID, msg, deps)
			return
		}
	}

	// Not a command - check for filter matches
	// Normalize the incoming message for matching
	normalizedMsg := domain.NormalizeMessage(msg.Text)

	// Convert chat ID safely
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}

	// Find all matching filters for this chat
	filters, err := deps.FilterRepository.FindMatchingFilters(ctx, chatID, normalizedMsg)
	if err != nil {
		logger.Errorf("Failed to find matching filters for chat %d: %v", msg.ChatId, err)
		return
	}

	// Dispatch responses for each matching filter
	for _, filter := range filters {
		switch filter.ResponseType {
		case domain.ResponseTypeText:
			// Send text response as a quote reply
			_, err := rpc.MiscSendTextMessage(accID, msg.ChatId, filter.ResponseText)
			if err != nil {
				logger.Errorf("Failed to send text response to chat %d: %v", msg.ChatId, err)
				continue
			}
			// Set the message as a quote reply
			// Note: Delta Chat RPC doesn't have direct quote API in MiscSendTextMessage,
			// we'll need to use the quote field in message data if available

		case domain.ResponseTypeMedia:
			// Read media file from storage
			mediaData, err := deps.MediaStorage.Read(filter.MediaHash)
			if err != nil {
				logger.Errorf("Failed to read media file %s: %v", filter.MediaHash, err)
				continue
			}

			// Send media (implementation depends on Delta Chat RPC media sending API)
			// This is a placeholder - actual implementation will need the proper RPC method
			logger.Infof("Would send media %s (%s) to chat %d in response to msg %d",
				filter.MediaHash, filter.MediaType, msg.ChatId, msgID)
			_ = mediaData // Use the data when proper API is available

		case domain.ResponseTypeReaction:
			// Send reaction to the triggering message
			_, err := rpc.SendReaction(accID, msgID, filter.Reaction)
			if err != nil {
				logger.Errorf("Failed to send reaction %s to message %d: %v", filter.Reaction, msgID, err)
				continue
			}

		default:
			logger.Errorf("Unknown response type %s for filter %d", filter.ResponseType, filter.FilterID)
		}
	}
}

// sendErrorMessage sends an error message to the chat and logs if sending fails.
func sendErrorMessage(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	message string,
) {
	if _, err := rpc.MiscSendTextMessage(accID, chatID, message); err != nil {
		logger.Errorf("Failed to send error message to chat %d: %v", chatID, err)
	}
}

// sendConfirmation sends a confirmation message to the chat and logs if sending fails.
func sendConfirmation(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	message string,
) {
	if _, err := rpc.MiscSendTextMessage(accID, chatID, message); err != nil {
		logger.Errorf("Failed to send confirmation to chat %d: %v", chatID, err)
	}
}

// validateAndNormalizeTriggers validates all triggers and returns normalized versions.
func validateAndNormalizeTriggers(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	triggers []string,
) ([]string, bool) {
	// Validate all triggers
	for _, trigger := range triggers {
		if err := domain.ValidateTrigger(trigger); err != nil {
			errMsg := fmt.Sprintf("❌ Invalid trigger '%s': %v", trigger, err)
			sendErrorMessage(rpc, logger, accID, chatID, errMsg)
			return nil, false
		}
	}

	// Normalize triggers for storage
	normalized := make([]string, len(triggers))
	for i, trigger := range triggers {
		normalized[i] = domain.NormalizeTrigger(trigger)
	}

	return normalized, true
}

// handleTextFilterCreation creates a text filter and sends confirmation.
func handleTextFilterCreation(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	dbChatID int64,
	cmd *domain.FilterCommand,
	normalizedTriggers []string,
	isMediaFilter bool,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// If replying to a media message but response is text, that's probably an error
	if isMediaFilter {
		errMsg := "❌ You're replying to a media message. Did you mean to create a media filter? Remove the reply or use the command without text if you want to create a media filter."
		sendErrorMessage(rpc, logger, accID, chatID, errMsg)
		return
	}

	err := deps.FilterRepository.CreateTextFilter(ctx, dbChatID, normalizedTriggers, cmd.ResponseText)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to create filter: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, errMsg)
		return
	}

	// Send confirmation
	triggerList := strings.Join(cmd.Triggers, ", ")
	confirmMsg := fmt.Sprintf("✅ Filter created! Triggers: %s", triggerList)
	sendConfirmation(rpc, logger, accID, chatID, confirmMsg)
}

// handleReactionFilterCreation creates a reaction filter and sends confirmation.
func handleReactionFilterCreation(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	dbChatID int64,
	cmd *domain.FilterCommand,
	normalizedTriggers []string,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	err := deps.FilterRepository.CreateReactionFilter(ctx, dbChatID, normalizedTriggers, cmd.Reaction)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to create reaction filter: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, errMsg)
		return
	}

	// Send confirmation
	triggerList := strings.Join(cmd.Triggers, ", ")
	confirmMsg := fmt.Sprintf("✅ Reaction filter created! Triggers: %s → %s", triggerList, cmd.Reaction)
	sendConfirmation(rpc, logger, accID, chatID, confirmMsg)
}

// downloadMediaIfNeeded ensures the quoted message media is downloaded and returns the updated message.
func downloadMediaIfNeeded(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	quotedMsg *deltachat.MsgSnapshot,
	quotedMsgID deltachat.MsgId,
) (*deltachat.MsgSnapshot, error) {
	if quotedMsg.DownloadState == deltachat.DownloadDone {
		return quotedMsg, nil
	}

	if quotedMsg.DownloadState != deltachat.DownloadAvailable {
		sendErrorMessage(rpc, logger, accID, chatID, "❌ Media is not available for download.")
		return nil, fmt.Errorf("media not available")
	}

	// Try to download it
	if err := rpc.DownloadFullMessage(accID, quotedMsgID); err != nil {
		errMsg := fmt.Sprintf("❌ Failed to download media: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, errMsg)
		return nil, err
	}

	// Re-fetch the message after download
	updatedMsg, err := rpc.GetMessage(accID, quotedMsgID)
	if err != nil || updatedMsg.DownloadState != deltachat.DownloadDone {
		sendErrorMessage(rpc, logger, accID, chatID, "❌ Media download incomplete. Please try again in a moment.")
		return nil, fmt.Errorf("download incomplete")
	}

	return updatedMsg, nil
}

// processMediaFile reads the media file, computes its hash, and saves it to storage.
func processMediaFile(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	filePath string,
	deps *domain.Dependencies,
) (string, error) {
	if filePath == "" {
		sendErrorMessage(rpc, logger, accID, chatID, "❌ No file path found in media message.")
		return "", fmt.Errorf("no file path")
	}

	// #nosec G304 -- filePath comes from Delta Chat RPC API (quotedMsg.File), not user input
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to read media file: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, errMsg)
		return "", err
	}

	// Compute SHA-512 hash
	hash := sha512.Sum512(fileData)
	mediaHash := hex.EncodeToString(hash[:])

	// Save to media storage
	if err := deps.MediaStorage.Save(mediaHash, fileData); err != nil {
		errMsg := fmt.Sprintf("❌ Failed to save media file: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, errMsg)
		return "", err
	}

	return mediaHash, nil
}

// handleMediaFilterCreation creates a media filter from an attached or quoted media message.
func handleMediaFilterCreation(
	rpc rpcClient,
	logger interface{ Errorf(string, ...interface{}) },
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	dbChatID int64,
	cmd *domain.FilterCommand,
	normalizedTriggers []string,
	hasQuotedMedia bool,
	hasAttachedMedia bool,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	if !hasAttachedMedia && !hasQuotedMedia {
		errMsg := "❌ To create a media filter, attach media (image, sticker, GIF, or video) to the /filter command, or reply to a media message."
		sendErrorMessage(rpc, logger, accID, msg.ChatId, errMsg)
		return
	}

	// Resolve the message that carries the media.
	// Prefer an attachment on the current message; fall back to a quoted message.
	var mediaMsg *deltachat.MsgSnapshot
	if hasAttachedMedia {
		mediaMsg = msg
	} else {
		quotedMsg, err := rpc.GetMessage(accID, msg.Quote.MessageId)
		if err != nil {
			errMsg := fmt.Sprintf("❌ Failed to get quoted message: %v", err)
			sendErrorMessage(rpc, logger, accID, msg.ChatId, errMsg)
			return
		}

		// Check if the quoted message is a supported media type
		if mapViewTypeToMediaType(quotedMsg.ViewType) == "" {
			errMsg := fmt.Sprintf("❌ Unsupported media type: %s. Supported types: image, sticker, gif, video.", quotedMsg.ViewType)
			sendErrorMessage(rpc, logger, accID, msg.ChatId, errMsg)
			return
		}

		// Ensure media is downloaded
		quotedMsg, err = downloadMediaIfNeeded(rpc, logger, accID, msg.ChatId, quotedMsg, msg.Quote.MessageId)
		if err != nil {
			return // Error already sent to user
		}
		mediaMsg = quotedMsg
	}

	mediaType := mapViewTypeToMediaType(mediaMsg.ViewType)

	// Process the media file (read, hash, save)
	mediaHash, err := processMediaFile(rpc, logger, accID, msg.ChatId, mediaMsg.File, deps)
	if err != nil {
		return // Error already sent to user
	}

	// Create media filter
	err = deps.FilterRepository.CreateMediaFilter(ctx, dbChatID, normalizedTriggers, mediaHash, mediaType)
	if err != nil {
		// Clean up the saved file
		_ = deps.MediaStorage.Delete(mediaHash)
		errMsg := fmt.Sprintf("❌ Failed to create media filter: %v", err)
		sendErrorMessage(rpc, logger, accID, msg.ChatId, errMsg)
		return
	}

	// Send confirmation
	triggerList := strings.Join(cmd.Triggers, ", ")
	confirmMsg := fmt.Sprintf("✅ Media filter created! Triggers: %s → [%s]", triggerList, mediaType)
	sendConfirmation(rpc, logger, accID, msg.ChatId, confirmMsg)
}

// handleFilterCommand processes a /filter command
func handleFilterCommand(
	rpc rpcClient,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	// Convert chat ID safely
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}

	// Parse the command
	cmd, err := domain.ParseFilterCommand(msg.Text)
	if err != nil {
		sendErrorMessage(rpc, logger, accID, msg.ChatId, "❌ Invalid command syntax. "+err.Error())
		return
	}

	// Validate and normalize triggers
	normalizedTriggers, ok := validateAndNormalizeTriggers(rpc, logger, accID, msg.ChatId, cmd.Triggers)
	if !ok {
		return // Error already sent to user
	}

	// Detect where media might come from: a quoted message or an attachment on this message.
	hasQuotedMedia := msg.Quote != nil && msg.Quote.MessageId != 0
	hasAttachedMedia := mapViewTypeToMediaType(msg.ViewType) != "" && msg.File != ""
	hasMedia := hasQuotedMedia || hasAttachedMedia

	// Handle based on response type
	switch cmd.ResponseType {
	case domain.ResponseTypeText:
		handleTextFilterCreation(rpc, logger, accID, msg.ChatId, chatID, cmd, normalizedTriggers, hasMedia, deps)
	case domain.ResponseTypeReaction:
		handleReactionFilterCreation(rpc, logger, accID, msg.ChatId, chatID, cmd, normalizedTriggers, deps)
	case domain.ResponseTypeMedia:
		handleMediaFilterCreation(rpc, logger, accID, msg, chatID, cmd, normalizedTriggers, hasQuotedMedia, hasAttachedMedia, deps)
	}
}

// mapViewTypeToMediaType maps Delta Chat view types to our media type constants
func mapViewTypeToMediaType(viewType deltachat.MsgType) string {
	switch viewType {
	case deltachat.MsgImage:
		return domain.MediaTypeImage
	case deltachat.MsgSticker:
		return domain.MediaTypeSticker
	case deltachat.MsgGif:
		return domain.MediaTypeGIF
	case deltachat.MsgVideo:
		return domain.MediaTypeVideo
	default:
		return ""
	}
}

// handleStopCommand processes a /stop command
func handleStopCommand(
	rpc rpcClient,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Convert chat ID safely
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}

	// Parse the command
	cmd, err := domain.ParseStopCommand(msg.Text)
	if err != nil {
		if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, "❌ Invalid command syntax. "+err.Error()); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Normalize trigger for lookup
	normalizedTrigger := domain.NormalizeTrigger(cmd.Trigger)

	// Remove the trigger
	mediaHash, err := deps.FilterRepository.RemoveTrigger(ctx, chatID, normalizedTrigger)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to remove trigger '%s': %v", cmd.Trigger, err)
		if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Clean up media file if this was a media filter
	if mediaHash != nil {
		if err := deps.MediaStorage.Delete(*mediaHash); err != nil {
			logger.Errorf("Failed to delete media file %s: %v", *mediaHash, err)
			// Continue anyway - the filter is already removed from the database
		}
	}

	// Send confirmation
	confirmMsg := fmt.Sprintf("✅ Trigger '%s' removed", cmd.Trigger)
	if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
		logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
	}
}

// handleStopAllCommand processes a /stopall command
func handleStopAllCommand(
	rpc rpcClient,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Convert chat ID safely
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}

	// Remove all filters for this chat
	mediaHashes, err := deps.FilterRepository.RemoveAllFilters(ctx, chatID)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to remove filters: %v", err)
		if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Clean up media files
	for _, hash := range mediaHashes {
		if err := deps.MediaStorage.Delete(hash); err != nil {
			logger.Errorf("Failed to delete media file %s: %v", hash, err)
			// Continue anyway - the filter is already removed from the database
		}
	}

	// Send confirmation
	confirmMsg := "✅ All filters removed from this chat"
	if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
		logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
	}
}

// handleFiltersCommand processes a /filters command
func handleFiltersCommand(
	rpc rpcClient,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Convert chat ID safely
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}

	// Get all filters for this chat
	filters, err := deps.FilterRepository.ListFilters(ctx, chatID)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to list filters: %v", err)
		if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Check if there are no filters
	if len(filters) == 0 {
		if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, "No filters configured for this chat."); sendErr != nil {
			logger.Errorf("Failed to send message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Build the filter list message
	var sb strings.Builder
	sb.WriteString("📋 Active filters:\n\n")

	for _, filter := range filters {
		// Format triggers
		triggerTexts := make([]string, len(filter.Triggers))
		for i, trigger := range filter.Triggers {
			triggerTexts[i] = trigger.TriggerText
		}
		triggerList := strings.Join(triggerTexts, ", ")

		// Format response based on type
		var responseDesc string
		switch filter.ResponseType {
		case domain.ResponseTypeText:
			// Truncate long responses
			respText := filter.Response.ResponseText
			if len(respText) > 50 {
				respText = respText[:47] + "..."
			}
			responseDesc = fmt.Sprintf("→ %s", respText)
		case domain.ResponseTypeReaction:
			responseDesc = fmt.Sprintf("→ %s", filter.Response.Reaction)
		case domain.ResponseTypeMedia:
			responseDesc = fmt.Sprintf("→ [%s media]", filter.Response.MediaType)
		default:
			responseDesc = fmt.Sprintf("→ [%s]", filter.ResponseType)
		}

		sb.WriteString(fmt.Sprintf("• %s %s\n", triggerList, responseDesc))
	}

	// Send the list
	if _, sendErr := rpc.MiscSendTextMessage(accID, msg.ChatId, sb.String()); sendErr != nil {
		logger.Errorf("Failed to send filter list to chat %d: %v", msg.ChatId, sendErr)
	}
}

// handleDMMessage processes a direct message by replying with help text.
func handleDMMessage(rpc rpcClient, logger interface{ Errorf(string, ...interface{}) }, accID deltachat.AccountId, msg *deltachat.MsgSnapshot) {
	if _, err := rpc.MiscSendTextMessage(accID, msg.ChatId, helpText); err != nil {
		logger.Errorf("Failed to send help text to chat %d: %v", msg.ChatId, err)
	}
}
