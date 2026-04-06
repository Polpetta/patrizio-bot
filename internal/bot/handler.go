package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"

	"github.com/polpetta/patrizio/internal/domain"
)

type handlerLogger interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
	Warnf(string, ...interface{})
}

const helpText = `Hi! I'm Patrizio, a group chat bot.

Add me to a group and I'll respond to messages based on configured filters. Here's what I can do:

/filter <trigger> <response>
  Create a filter. When a message contains the trigger word, I'll reply with the response text.
  Examples:
  /filter hello Hi there!
  /filter "good morning" Rise and shine!

/filter (<trigger1>, <trigger2>, ...) <response>
  Create a filter with multiple triggers for the same response.
  Example:
  /filter (hi, hello, "good morning") Hey!

/filter <trigger> react:<emoji>
  Create a reaction filter. I'll react to the triggering message with the given emoji.
  Example:
  /filter lol react:😂

/filter <trigger>
  Create a media filter. Attach an image, sticker, GIF, or video to the command, or reply to a media message. I'll send that media when the trigger matches.
  Example:
  /filter cat (with an image attached)

/stop <trigger>
  Remove a single trigger.
  Examples:
  /stop hello
  /stop "good morning"

/stopall
  Remove all filters from the current chat.

/filters
  List all active filters in the current chat.

/prompt <message>
  Send a message to the AI assistant. Each /prompt starts a new conversation thread.
  To continue the conversation, simply reply to my response — no need to type /prompt again.
  Example:
  /prompt What is the capital of France?

Triggers are matched as whole words anywhere in a message and are case-insensitive.`

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
		go processMessage(bot.Rpc, cli.GetLogger(accID), accID, msgID, deps)
	}
}

// processMessage contains the core per-message routing logic.
// It is extracted from the newMsgHandler goroutine so it can be called
// synchronously in tests without depending on *botcli.BotCli.
func processMessage(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
	deps *domain.Dependencies,
) {
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

	logger.Infof("Received message %d in chat %d (type: %s)", msgID, msg.ChatId, chatInfo.ChatType)
	switch chatInfo.ChatType {
	case deltachat.ChatGroup, deltachat.ChatOutBroadcast, deltachat.ChatInBroadcast, deltachat.ChatMailinglist:
		handleGroupMessage(rpc, logger, accID, msgID, msg, deps)
	case deltachat.ChatSingle:
		handleDMMessage(rpc, logger, accID, msgID, msg, deps)
	default:
		logger.Warnf("Unknown chat type %s for chat %d, ignoring", chatInfo.ChatType, msg.ChatId)
	}
}

// handleGroupMessage processes a message from a group chat.
// It checks for commands first, then normalizes the message and checks for matching filters.
func handleGroupMessage(
	rpc rpcClient,
	logger handlerLogger,
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
			handleFilterCommand(rpc, logger, accID, msgID, msg, deps)
			return
		case domain.CommandStop:
			handleStopCommand(rpc, logger, accID, msgID, msg, deps)
			return
		case domain.CommandStopAll:
			handleStopAllCommand(rpc, logger, accID, msgID, msg, deps)
			return
		case domain.CommandFilters:
			handleFiltersCommand(rpc, logger, accID, msgID, msg, deps)
			return
		case domain.CommandPrompt:
			handlePromptCommand(rpc, logger, accID, msgID, msg, deps)
			return
		}
	}

	// Check for thread continuation (reply to a Patrizio conversation message)
	if isContinuation, threadRootID := isThreadContinuation(ctx, msg, deps); isContinuation {
		handleThreadContinuation(rpc, logger, accID, msgID, msg, deps, threadRootID)
		return
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
			// Send text response as a quote-reply to the triggering message
			_, err := rpc.SendMsg(accID, msg.ChatId, deltachat.MsgData{
				Text:            filter.ResponseText,
				QuotedMessageId: msgID,
			})
			if err != nil {
				logger.Errorf("Failed to send text response to chat %d: %v", msg.ChatId, err)
				continue
			}

		case domain.ResponseTypeMedia:
			// Look up the media file path from storage
			mediaPath := deps.MediaStorage.Path(filter.MediaHash)
			exists, err := deps.MediaStorage.Exists(filter.MediaHash)
			if err != nil || !exists {
				logger.Errorf("Media file %s not found in storage", filter.MediaHash)
				continue
			}

			// Map domain media type back to Delta Chat view type
			viewType := mapMediaTypeToViewType(filter.MediaType)
			if viewType == "" {
				logger.Errorf("Unknown media type %s for filter %d", filter.MediaType, filter.FilterID)
				continue
			}

			// Send the media message as a quote-reply to the triggering message
			_, err = rpc.SendMsg(accID, msg.ChatId, deltachat.MsgData{
				File:            mediaPath,
				ViewType:        viewType,
				QuotedMessageId: msgID,
			})
			if err != nil {
				logger.Errorf("Failed to send media response to chat %d: %v", msg.ChatId, err)
				continue
			}

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

// sendErrorMessage sends an error message as a quote-reply and logs if sending fails.
func sendErrorMessage(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
	message string,
) {
	if _, err := rpc.SendMsg(accID, chatID, deltachat.MsgData{
		Text:            message,
		QuotedMessageId: replyTo,
	}); err != nil {
		logger.Errorf("Failed to send error message to chat %d: %v", chatID, err)
	}
}

// sendConfirmation sends a confirmation message as a quote-reply and logs if sending fails.
func sendConfirmation(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
	message string,
) {
	if _, err := rpc.SendMsg(accID, chatID, deltachat.MsgData{
		Text:            message,
		QuotedMessageId: replyTo,
	}); err != nil {
		logger.Errorf("Failed to send confirmation to chat %d: %v", chatID, err)
	}
}

// validateAndNormalizeTriggers validates all triggers and returns normalized versions.
func validateAndNormalizeTriggers(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
	triggers []string,
) ([]string, bool) {
	// Validate all triggers
	for _, trigger := range triggers {
		if err := domain.ValidateTrigger(trigger); err != nil {
			errMsg := fmt.Sprintf("❌ Invalid trigger '%s': %v", trigger, err)
			sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
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
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
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
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
		return
	}

	err := deps.FilterRepository.CreateTextFilter(ctx, dbChatID, normalizedTriggers, cmd.ResponseText)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to create filter: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
		return
	}

	// Send confirmation
	triggerList := strings.Join(cmd.Triggers, ", ")
	confirmMsg := fmt.Sprintf("✅ Filter created! Triggers: %s", triggerList)
	sendConfirmation(rpc, logger, accID, chatID, replyTo, confirmMsg)
}

// handleReactionFilterCreation creates a reaction filter and sends confirmation.
func handleReactionFilterCreation(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
	dbChatID int64,
	cmd *domain.FilterCommand,
	normalizedTriggers []string,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	err := deps.FilterRepository.CreateReactionFilter(ctx, dbChatID, normalizedTriggers, cmd.Reaction)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to create reaction filter: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
		return
	}

	// Send confirmation
	triggerList := strings.Join(cmd.Triggers, ", ")
	confirmMsg := fmt.Sprintf("✅ Reaction filter created! Triggers: %s → %s", triggerList, cmd.Reaction)
	sendConfirmation(rpc, logger, accID, chatID, replyTo, confirmMsg)
}

// downloadMediaIfNeeded ensures the quoted message media is downloaded and returns the updated message.
func downloadMediaIfNeeded(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
	quotedMsg *deltachat.MsgSnapshot,
	quotedMsgID deltachat.MsgId,
) (*deltachat.MsgSnapshot, error) {
	if quotedMsg.DownloadState == deltachat.DownloadDone {
		return quotedMsg, nil
	}

	if quotedMsg.DownloadState != deltachat.DownloadAvailable {
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, "❌ Media is not available for download.")
		return nil, fmt.Errorf("media not available")
	}

	// Try to download it
	if err := rpc.DownloadFullMessage(accID, quotedMsgID); err != nil {
		errMsg := fmt.Sprintf("❌ Failed to download media: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
		return nil, err
	}

	// Re-fetch the message after download
	updatedMsg, err := rpc.GetMessage(accID, quotedMsgID)
	if err != nil || updatedMsg.DownloadState != deltachat.DownloadDone {
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, "❌ Media download incomplete. Please try again in a moment.")
		return nil, fmt.Errorf("download incomplete")
	}

	return updatedMsg, nil
}

// processMediaFile reads the media file, computes its hash, and saves it to storage.
func processMediaFile(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	chatID deltachat.ChatId,
	replyTo deltachat.MsgId,
	filePath string,
	deps *domain.Dependencies,
) (string, error) {
	if filePath == "" {
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, "❌ No file path found in media message.")
		return "", fmt.Errorf("no file path")
	}

	// #nosec G304 -- filePath comes from Delta Chat RPC API (quotedMsg.File), not user input
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to read media file: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
		return "", err
	}

	// Compute SHA-512 hash and preserve original file extension
	hash := sha512.Sum512(fileData)
	mediaHash := hex.EncodeToString(hash[:]) + filepath.Ext(filePath)

	// Save to media storage
	if err := deps.MediaStorage.Save(mediaHash, fileData); err != nil {
		errMsg := fmt.Sprintf("❌ Failed to save media file: %v", err)
		sendErrorMessage(rpc, logger, accID, chatID, replyTo, errMsg)
		return "", err
	}

	return mediaHash, nil
}

// handleMediaFilterCreation creates a media filter from an attached or quoted media message.
func handleMediaFilterCreation(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	replyTo deltachat.MsgId,
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
		sendErrorMessage(rpc, logger, accID, msg.ChatId, replyTo, errMsg)
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
			sendErrorMessage(rpc, logger, accID, msg.ChatId, replyTo, errMsg)
			return
		}

		// Check if the quoted message is a supported media type
		if mapViewTypeToMediaType(quotedMsg.ViewType) == "" {
			errMsg := fmt.Sprintf("❌ Unsupported media type: %s. Supported types: image, sticker, gif, video.", quotedMsg.ViewType)
			sendErrorMessage(rpc, logger, accID, msg.ChatId, replyTo, errMsg)
			return
		}

		// Ensure media is downloaded
		quotedMsg, err = downloadMediaIfNeeded(rpc, logger, accID, msg.ChatId, replyTo, quotedMsg, msg.Quote.MessageId)
		if err != nil {
			return // Error already sent to user
		}
		mediaMsg = quotedMsg
	}

	mediaType := mapViewTypeToMediaType(mediaMsg.ViewType)

	// Process the media file (read, hash, save)
	mediaHash, err := processMediaFile(rpc, logger, accID, msg.ChatId, replyTo, mediaMsg.File, deps)
	if err != nil {
		return // Error already sent to user
	}

	// Create media filter
	err = deps.FilterRepository.CreateMediaFilter(ctx, dbChatID, normalizedTriggers, mediaHash, mediaType)
	if err != nil {
		// Clean up the saved file
		if nestedErr := deps.MediaStorage.Delete(mediaHash); nestedErr != nil {
			logger.Errorf("Failed to delete media file %s after filter creation error: %v", mediaHash, nestedErr)
		}
		errMsg := fmt.Sprintf("❌ Failed to create media filter: %v", err)
		sendErrorMessage(rpc, logger, accID, msg.ChatId, replyTo, errMsg)
		return
	}

	// Send confirmation
	triggerList := strings.Join(cmd.Triggers, ", ")
	confirmMsg := fmt.Sprintf("✅ Media filter created! Triggers: %s → [%s]", triggerList, mediaType)
	sendConfirmation(rpc, logger, accID, msg.ChatId, replyTo, confirmMsg)
}

// handleFilterCommand processes a /filter command
func handleFilterCommand(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
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
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID, "❌ Invalid command syntax. "+err.Error())
		return
	}

	// Validate and normalize triggers
	normalizedTriggers, ok := validateAndNormalizeTriggers(rpc, logger, accID, msg.ChatId, msgID, cmd.Triggers)
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
		handleTextFilterCreation(rpc, logger, accID, msg.ChatId, msgID, chatID, cmd, normalizedTriggers, hasMedia, deps)
	case domain.ResponseTypeReaction:
		handleReactionFilterCreation(rpc, logger, accID, msg.ChatId, msgID, chatID, cmd, normalizedTriggers, deps)
	case domain.ResponseTypeMedia:
		handleMediaFilterCreation(rpc, logger, accID, msg, msgID, chatID, cmd, normalizedTriggers, hasQuotedMedia, hasAttachedMedia, deps)
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

// mapMediaTypeToViewType maps our media type constants back to Delta Chat view types
func mapMediaTypeToViewType(mediaType string) deltachat.MsgType {
	switch mediaType {
	case domain.MediaTypeImage:
		return deltachat.MsgImage
	case domain.MediaTypeSticker:
		return deltachat.MsgSticker
	case domain.MediaTypeGIF:
		return deltachat.MsgGif
	case domain.MediaTypeVideo:
		return deltachat.MsgVideo
	default:
		return ""
	}
}

// handleStopCommand processes a /stop command
func handleStopCommand(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
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
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID, "❌ Invalid command syntax. "+err.Error())
		return
	}

	// Normalize trigger for lookup
	normalizedTrigger := domain.NormalizeTrigger(cmd.Trigger)

	// Remove the trigger
	mediaHash, err := deps.FilterRepository.RemoveTrigger(ctx, chatID, normalizedTrigger)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to remove trigger '%s': %v", cmd.Trigger, err)
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID, errMsg)
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
	sendConfirmation(rpc, logger, accID, msg.ChatId, msgID, confirmMsg)
}

// handleStopAllCommand processes a /stopall command
func handleStopAllCommand(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
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
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID, errMsg)
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
	sendConfirmation(rpc, logger, accID, msg.ChatId, msgID, "✅ All filters removed from this chat")
}

// handleFiltersCommand processes a /filters command
func handleFiltersCommand(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
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
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID, errMsg)
		return
	}

	// Check if there are no filters
	if len(filters) == 0 {
		sendConfirmation(rpc, logger, accID, msg.ChatId, msgID, "No filters configured for this chat.")
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

		fmt.Fprintf(&sb, "• %s %s\n", triggerList, responseDesc)
	}

	// Send the list
	sendConfirmation(rpc, logger, accID, msg.ChatId, msgID, sb.String())
}

// isAllowedChat checks if the given chat ID is in the allowlist.
// Returns true if the allowlist is empty (open access) or if the chat ID is in the list.
func isAllowedChat(chatID int64, deps *domain.Dependencies) bool {
	allowedChatIDs := deps.Config.OpenAIAllowedChatIDs()
	if len(allowedChatIDs) == 0 {
		return true // Empty list means all chats are allowed
	}
	for _, id := range allowedChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

// isThreadContinuation checks if the message is a reply to a Patrizio conversation message.
// Returns (true, threadRootID) if it's a continuation, (false, 0) otherwise.
func isThreadContinuation(ctx context.Context, msg *deltachat.MsgSnapshot, deps *domain.Dependencies) (bool, int64) {
	if deps.ConversationRepository == nil {
		return false, 0
	}
	if msg.Quote == nil || msg.Quote.MessageId == 0 {
		return false, 0
	}

	//nolint:gosec // G115: MsgId conversion is safe — Delta Chat MsgIds are small positive integers
	quotedMsgID := int64(msg.Quote.MessageId)
	exists, threadRootID, err := deps.ConversationRepository.IsConversationMessage(ctx, quotedMsgID)
	if err != nil || !exists {
		return false, 0
	}

	return true, *threadRootID
}

// handlePromptCommand processes a /prompt command, creating a new conversation thread.
func handlePromptCommand(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Check if AI client is configured
	if deps.AIClient == nil {
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"The AI assistant is not configured. Please set the proper configuration (API key, base URL, model) and restart the bot.")
		return
	}

	// Check allowlist
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}
	if !isAllowedChat(chatID, deps) {
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"This chat is not authorized to use the AI assistant.")
		return
	}

	// Parse the prompt message
	promptText, err := domain.ParsePromptCommand(msg.Text)
	if err != nil {
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID, fmt.Sprintf("Invalid command: %v", err))
		return
	}

	// Build message array: system prompt (if set) + user message
	var messages []domain.ChatMessage
	if sysPrompt := deps.Config.OpenAISystemPrompt(); sysPrompt != "" {
		messages = append(messages, domain.ChatMessage{Role: "system", Content: sysPrompt})
	}
	messages = append(messages, domain.ChatMessage{Role: "user", Content: promptText})

	// Call AI client
	response, err := deps.AIClient.ChatCompletion(ctx, messages)
	if err != nil {
		logger.Errorf("AI completion failed for chat %d: %v", msg.ChatId, err)
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"Sorry, I encountered an error while processing your request. Please try again later.")
		return
	}

	// Send response as quote-reply
	responseMsgID, err := rpc.SendMsg(accID, msg.ChatId, deltachat.MsgData{
		Text:            response,
		QuotedMessageId: msgID,
	})
	if err != nil {
		logger.Errorf("Failed to send AI response to chat %d: %v", msg.ChatId, err)
		return
	}

	// Save both messages to conversation repository.
	// The thread root is the user's message (the /prompt message).
	//nolint:gosec // G115: MsgId conversion is safe — Delta Chat MsgIds are small positive integers
	userMsgID := int64(msgID)
	//nolint:gosec // G115: MsgId conversion is safe — Delta Chat MsgIds are small positive integers
	assistantMsgID := int64(responseMsgID)

	// Save user message (root of thread, no parent)
	if err := deps.ConversationRepository.SaveMessage(ctx, userMsgID, userMsgID, nil, "user", promptText); err != nil {
		logger.Errorf("Failed to save user message: %v", err)
	}

	// Save assistant message (parent is user message)
	if err := deps.ConversationRepository.SaveMessage(ctx, userMsgID, assistantMsgID, &userMsgID, "assistant", response); err != nil {
		logger.Errorf("Failed to save assistant message: %v", err)
	}
}

// handleThreadContinuation processes a reply to a Patrizio conversation message,
// continuing an existing thread.
func handleThreadContinuation(
	rpc rpcClient,
	logger handlerLogger,
	accID deltachat.AccountId,
	msgID deltachat.MsgId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
	threadRootID int64,
) {
	ctx := context.Background()

	// Check if AI client is configured
	if deps.AIClient == nil {
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"The AI assistant is not configured. Please set the proper configuration (API key, base URL, model) and restart the bot.")
		return
	}

	// Check allowlist
	chatID, err := convertChatID(msg.ChatId)
	if err != nil {
		logger.Errorf("Invalid chat ID %d: %v", msg.ChatId, err)
		return
	}
	if !isAllowedChat(chatID, deps) {
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"This chat is not authorized to use the AI assistant.")
		return
	}

	// Get the quoted message's MsgId (the leaf of the existing chain)
	//nolint:gosec // G115: MsgId conversion is safe — Delta Chat MsgIds are small positive integers
	quotedMsgID := int64(msg.Quote.MessageId)

	// Retrieve the existing conversation chain
	maxHistory := deps.Config.OpenAIMaxHistory()
	chain, err := deps.ConversationRepository.GetThreadChain(ctx, quotedMsgID, maxHistory)
	if err != nil {
		logger.Errorf("Failed to get thread chain: %v", err)
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"Sorry, I encountered an error while retrieving the conversation history.")
		return
	}

	// Build message array: system prompt + chain + new user message
	var messages []domain.ChatMessage
	if sysPrompt := deps.Config.OpenAISystemPrompt(); sysPrompt != "" {
		messages = append(messages, domain.ChatMessage{Role: "system", Content: sysPrompt})
	}
	messages = append(messages, chain...)
	messages = append(messages, domain.ChatMessage{Role: "user", Content: msg.Text})

	// Call AI client
	response, err := deps.AIClient.ChatCompletion(ctx, messages)
	if err != nil {
		logger.Errorf("AI completion failed for chat %d: %v", msg.ChatId, err)
		sendErrorMessage(rpc, logger, accID, msg.ChatId, msgID,
			"Sorry, I encountered an error while processing your request. Please try again later.")
		return
	}

	// Send response as quote-reply
	responseMsgID, err := rpc.SendMsg(accID, msg.ChatId, deltachat.MsgData{
		Text:            response,
		QuotedMessageId: msgID,
	})
	if err != nil {
		logger.Errorf("Failed to send AI response to chat %d: %v", msg.ChatId, err)
		return
	}

	// Save both messages to conversation repository
	//nolint:gosec // G115: MsgId conversion is safe — Delta Chat MsgIds are small positive integers
	userMsgID := int64(msgID)
	//nolint:gosec // G115: MsgId conversion is safe — Delta Chat MsgIds are small positive integers
	assistantMsgID := int64(responseMsgID)

	// Save user message (parent is the quoted message)
	if err := deps.ConversationRepository.SaveMessage(ctx, threadRootID, userMsgID, &quotedMsgID, "user", msg.Text); err != nil {
		logger.Errorf("Failed to save user message: %v", err)
	}

	// Save assistant message (parent is user message)
	if err := deps.ConversationRepository.SaveMessage(ctx, threadRootID, assistantMsgID, &userMsgID, "assistant", response); err != nil {
		logger.Errorf("Failed to save assistant message: %v", err)
	}
}

// handleDMMessage processes a direct message.
// It checks for /prompt command first, then thread continuation, then falls back to help text.
func handleDMMessage(rpc rpcClient, logger handlerLogger, accID deltachat.AccountId, msgID deltachat.MsgId, msg *deltachat.MsgSnapshot, deps *domain.Dependencies) {
	ctx := context.Background()

	// Check for /prompt command
	cmdType := domain.GetCommandType(msg.Text)
	if cmdType == domain.CommandPrompt {
		handlePromptCommand(rpc, logger, accID, msgID, msg, deps)
		return
	}

	// Check for thread continuation
	if isContinuation, threadRootID := isThreadContinuation(ctx, msg, deps); isContinuation {
		handleThreadContinuation(rpc, logger, accID, msgID, msg, deps, threadRootID)
		return
	}

	// Fall back to help text
	if _, err := rpc.MiscSendTextMessage(accID, msg.ChatId, helpText); err != nil {
		logger.Errorf("Failed to send help text to chat %d: %v", msg.ChatId, err)
	}
}
