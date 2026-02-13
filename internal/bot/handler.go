package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"

	"github.com/polpetta/patrizio/internal/domain"
)

const helpText = `Hi! I'm Patrizio, a group chat bot.

Add me to a group and I'll respond to messages based on configured filters.

I don't do much in direct messages — add me to a group to get started!`

// newMsgHandler returns the OnNewMsg callback that routes incoming messages.
func newMsgHandler(cli *botcli.BotCli, bot *deltachat.Bot, deps *domain.Dependencies) deltachat.NewMsgHandler {
	return func(bot *deltachat.Bot, accID deltachat.AccountId, msgID deltachat.MsgId) {
		logger := cli.GetLogger(accID)

		msg, err := bot.Rpc.GetMessage(accID, msgID)
		if err != nil {
			logger.Errorf("Failed to get message %d: %v", msgID, err)
			return
		}

		// Ignore messages from special contacts (system, device, etc.).
		if msg.FromId <= deltachat.ContactLastSpecial {
			return
		}

		chatInfo, err := bot.Rpc.GetBasicChatInfo(accID, msg.ChatId)
		if err != nil {
			logger.Errorf("Failed to get chat info for chat %d: %v", msg.ChatId, err)
			return
		}

		switch chatInfo.ChatType {
		case deltachat.ChatGroup, deltachat.ChatBroadcast, deltachat.ChatMailinglist:
			handleGroupMessage(bot, logger, accID, msgID, msg, deps)
		case deltachat.ChatSingle:
			handleDMMessage(bot, logger, accID, msg)
		default:
			logger.Warnf("Unknown chat type %d for chat %d, ignoring", chatInfo.ChatType, msg.ChatId)
		}
	}
}

// handleGroupMessage processes a message from a group chat.
// It checks for commands first, then normalizes the message and checks for matching filters.
func handleGroupMessage(
	bot *deltachat.Bot,
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
			handleFilterCommand(bot, logger, accID, msg, deps)
			return
		case domain.CommandStop:
			handleStopCommand(bot, logger, accID, msg, deps)
			return
		case domain.CommandStopAll:
			handleStopAllCommand(bot, logger, accID, msg, deps)
			return
		case domain.CommandFilters:
			handleFiltersCommand(bot, logger, accID, msg, deps)
			return
		}
	}

	// Not a command - check for filter matches
	// Normalize the incoming message for matching
	normalizedMsg := domain.NormalizeMessage(msg.Text)

	// Find all matching filters for this chat
	filters, err := deps.FilterRepository.FindMatchingFilters(ctx, int64(msg.ChatId), normalizedMsg)
	if err != nil {
		logger.Errorf("Failed to find matching filters for chat %d: %v", msg.ChatId, err)
		return
	}

	// Dispatch responses for each matching filter
	for _, filter := range filters {
		switch filter.ResponseType {
		case domain.ResponseTypeText:
			// Send text response as a quote reply
			_, err := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, filter.ResponseText)
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
			_, err := bot.Rpc.SendReaction(accID, msgID, filter.Reaction)
			if err != nil {
				logger.Errorf("Failed to send reaction %s to message %d: %v", filter.Reaction, msgID, err)
				continue
			}

		default:
			logger.Errorf("Unknown response type %s for filter %d", filter.ResponseType, filter.FilterID)
		}
	}
}

// handleFilterCommand processes a /filter command
func handleFilterCommand(
	bot *deltachat.Bot,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Parse the command
	cmd, err := domain.ParseFilterCommand(msg.Text)
	if err != nil {
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, "❌ Invalid command syntax. "+err.Error()); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Validate all triggers
	for _, trigger := range cmd.Triggers {
		if err := domain.ValidateTrigger(trigger); err != nil {
			errMsg := fmt.Sprintf("❌ Invalid trigger '%s': %v", trigger, err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}
	}

	// Normalize triggers for storage
	normalizedTriggers := make([]string, len(cmd.Triggers))
	for i, trigger := range cmd.Triggers {
		normalizedTriggers[i] = domain.NormalizeTrigger(trigger)
	}

	// Check if this is a reply to a media message
	isMediaFilter := msg.Quote != nil && msg.Quote.MessageId != 0

	// Handle based on response type
	switch cmd.ResponseType {
	case domain.ResponseTypeText:
		// If replying to a media message but response is text, that's probably an error
		if isMediaFilter {
			errMsg := "❌ You're replying to a media message. Did you mean to create a media filter? Remove the reply or use the command without text if you want to create a media filter."
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		err = deps.FilterRepository.CreateTextFilter(ctx, int64(msg.ChatId), normalizedTriggers, cmd.ResponseText)
		if err != nil {
			errMsg := fmt.Sprintf("❌ Failed to create filter: %v", err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Send confirmation
		triggerList := strings.Join(cmd.Triggers, ", ")
		confirmMsg := fmt.Sprintf("✅ Filter created! Triggers: %s", triggerList)
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
			logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
		}

	case domain.ResponseTypeReaction:
		err = deps.FilterRepository.CreateReactionFilter(ctx, int64(msg.ChatId), normalizedTriggers, cmd.Reaction)
		if err != nil {
			errMsg := fmt.Sprintf("❌ Failed to create reaction filter: %v", err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Send confirmation
		triggerList := strings.Join(cmd.Triggers, ", ")
		confirmMsg := fmt.Sprintf("✅ Reaction filter created! Triggers: %s → %s", triggerList, cmd.Reaction)
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
			logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
		}

	case domain.ResponseTypeMedia:
		// Media filters require a reply to a media message
		if !isMediaFilter {
			errMsg := "❌ To create a media filter, reply to a media message (image, sticker, GIF, or video) with the /filter command."
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Get the quoted message
		quotedMsg, err := bot.Rpc.GetMessage(accID, msg.Quote.MessageId)
		if err != nil {
			errMsg := fmt.Sprintf("❌ Failed to get quoted message: %v", err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Check if the quoted message is a supported media type
		mediaType := mapViewTypeToMediaType(quotedMsg.ViewType)
		if mediaType == "" {
			errMsg := fmt.Sprintf("❌ Unsupported media type: %s. Supported types: image, sticker, gif, video.", quotedMsg.ViewType)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Check if media is downloaded
		if quotedMsg.DownloadState != deltachat.DownloadDone {
			if quotedMsg.DownloadState == deltachat.DownloadAvailable {
				// Try to download it
				if err := bot.Rpc.DownloadFullMessage(accID, msg.Quote.MessageId); err != nil {
					errMsg := fmt.Sprintf("❌ Failed to download media: %v", err)
					if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
						logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
					}
					return
				}
				// Re-fetch the message after download
				quotedMsg, err = bot.Rpc.GetMessage(accID, msg.Quote.MessageId)
				if err != nil || quotedMsg.DownloadState != deltachat.DownloadDone {
					errMsg := "❌ Media download incomplete. Please try again in a moment."
					if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
						logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
					}
					return
				}
			} else {
				errMsg := "❌ Media is not available for download."
				if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
					logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
				}
				return
			}
		}

		// Read the media file
		if quotedMsg.File == "" {
			errMsg := "❌ No file path found in media message."
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		fileData, err := os.ReadFile(quotedMsg.File)
		if err != nil {
			errMsg := fmt.Sprintf("❌ Failed to read media file: %v", err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Compute SHA-512 hash
		hash := sha512.Sum512(fileData)
		mediaHash := hex.EncodeToString(hash[:])

		// Save to media storage
		if err := deps.MediaStorage.Save(mediaHash, fileData); err != nil {
			errMsg := fmt.Sprintf("❌ Failed to save media file: %v", err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Create media filter
		err = deps.FilterRepository.CreateMediaFilter(ctx, int64(msg.ChatId), normalizedTriggers, mediaHash, mediaType)
		if err != nil {
			// Clean up the saved file
			_ = deps.MediaStorage.Delete(mediaHash)
			errMsg := fmt.Sprintf("❌ Failed to create media filter: %v", err)
			if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
				logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
			}
			return
		}

		// Send confirmation
		triggerList := strings.Join(cmd.Triggers, ", ")
		confirmMsg := fmt.Sprintf("✅ Media filter created! Triggers: %s → [%s]", triggerList, mediaType)
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
			logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
		}
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
	bot *deltachat.Bot,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Parse the command
	cmd, err := domain.ParseStopCommand(msg.Text)
	if err != nil {
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, "❌ Invalid command syntax. "+err.Error()); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Normalize trigger for lookup
	normalizedTrigger := domain.NormalizeTrigger(cmd.Trigger)

	// Remove the trigger
	mediaHash, err := deps.FilterRepository.RemoveTrigger(ctx, int64(msg.ChatId), normalizedTrigger)
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to remove trigger '%s': %v", cmd.Trigger, err)
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
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
	if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
		logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
	}
}

// handleStopAllCommand processes a /stopall command
func handleStopAllCommand(
	bot *deltachat.Bot,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Remove all filters for this chat
	mediaHashes, err := deps.FilterRepository.RemoveAllFilters(ctx, int64(msg.ChatId))
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to remove filters: %v", err)
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
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
	if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, confirmMsg); sendErr != nil {
		logger.Errorf("Failed to send confirmation to chat %d: %v", msg.ChatId, sendErr)
	}
}

// handleFiltersCommand processes a /filters command
func handleFiltersCommand(
	bot *deltachat.Bot,
	logger interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
	accID deltachat.AccountId,
	msg *deltachat.MsgSnapshot,
	deps *domain.Dependencies,
) {
	ctx := context.Background()

	// Get all filters for this chat
	filters, err := deps.FilterRepository.ListFilters(ctx, int64(msg.ChatId))
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to list filters: %v", err)
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, errMsg); sendErr != nil {
			logger.Errorf("Failed to send error message to chat %d: %v", msg.ChatId, sendErr)
		}
		return
	}

	// Check if there are no filters
	if len(filters) == 0 {
		if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, "No filters configured for this chat."); sendErr != nil {
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
	if _, sendErr := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, sb.String()); sendErr != nil {
		logger.Errorf("Failed to send filter list to chat %d: %v", msg.ChatId, sendErr)
	}
}

// handleDMMessage processes a direct message by replying with help text.
func handleDMMessage(bot *deltachat.Bot, logger interface{ Errorf(string, ...interface{}) }, accID deltachat.AccountId, msg *deltachat.MsgSnapshot) {
	if _, err := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, helpText); err != nil {
		logger.Errorf("Failed to send help text to chat %d: %v", msg.ChatId, err)
	}
}
