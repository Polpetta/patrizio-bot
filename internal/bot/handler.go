package bot

import (
	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"
)

const helpText = `Hi! I'm Patrizio, a group chat bot.

Add me to a group and I'll respond to messages based on configured filters.

I don't do much in direct messages — add me to a group to get started!`

// newMsgHandler returns the OnNewMsg callback that routes incoming messages.
func newMsgHandler(cli *botcli.BotCli) deltachat.NewMsgHandler {
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
			handleGroupMessage(logger, accID, msgID, msg)
		case deltachat.ChatSingle:
			handleDMMessage(bot, logger, accID, msg)
		default:
			logger.Warnf("Unknown chat type %d for chat %d, ignoring", chatInfo.ChatType, msg.ChatId)
		}
	}
}

// handleGroupMessage processes a message from a group chat.
// Currently a no-op placeholder for the future filter engine.
func handleGroupMessage(logger interface{ Infof(string, ...interface{}) }, _ deltachat.AccountId, msgID deltachat.MsgId, msg *deltachat.MsgSnapshot) {
	logger.Infof("Group message received in chat %d from contact %d (msg %d)", msg.ChatId, msg.FromId, msgID)
}

// handleDMMessage processes a direct message by replying with help text.
func handleDMMessage(bot *deltachat.Bot, logger interface{ Errorf(string, ...interface{}) }, accID deltachat.AccountId, msg *deltachat.MsgSnapshot) {
	if _, err := bot.Rpc.MiscSendTextMessage(accID, msg.ChatId, helpText); err != nil {
		logger.Errorf("Failed to send help text to chat %d: %v", msg.ChatId, err)
	}
}
