// Package deltachat provides a Messenger adapter backed by the Delta Chat RPC client.
package deltachat

import (
	"fmt"

	dc "github.com/chatmail/rpc-client-go/deltachat"

	"github.com/polpetta/patrizio/internal/domain"
)

// Messenger implements domain.Messenger using the Delta Chat RPC client.
type Messenger struct {
	rpc *dc.Rpc
}

// New creates a new Messenger adapter wrapping the given Delta Chat RPC client.
func New(rpc *dc.Rpc) *Messenger {
	return &Messenger{rpc: rpc}
}

// FetchMessage retrieves a message by ID and returns it as a domain IncomingMessage.
func (m *Messenger) FetchMessage(accountID uint64, msgID uint64) (*domain.IncomingMessage, error) {
	snapshot, err := m.rpc.GetMessage(dc.AccountId(accountID), dc.MsgId(msgID))
	if err != nil {
		return nil, err
	}

	msg := &domain.IncomingMessage{
		ID:            uint64(snapshot.Id),
		ChatID:        uint64(snapshot.ChatId),
		FromID:        uint64(snapshot.FromId),
		Text:          snapshot.Text,
		File:          snapshot.File,
		MediaType:     mapViewTypeToMediaType(snapshot.ViewType),
		DownloadState: mapDownloadState(snapshot.DownloadState),
	}

	if snapshot.Quote != nil && snapshot.Quote.MessageId != 0 {
		msg.Quote = &domain.QuotedMessage{
			MessageID: uint64(snapshot.Quote.MessageId),
		}
	}

	return msg, nil
}

// FetchChatType retrieves the chat type for a given chat.
func (m *Messenger) FetchChatType(accountID uint64, chatID uint64) (domain.ChatType, error) {
	info, err := m.rpc.GetBasicChatInfo(dc.AccountId(accountID), dc.ChatId(chatID))
	if err != nil {
		return "", err
	}

	switch info.ChatType {
	case dc.ChatGroup, dc.ChatOutBroadcast, dc.ChatInBroadcast, dc.ChatMailinglist:
		return domain.ChatTypeGroup, nil
	case dc.ChatSingle:
		return domain.ChatTypeSingle, nil
	default:
		return "", fmt.Errorf("unknown chat type: %s", info.ChatType)
	}
}

// SendTextReply sends a text message as a quote-reply and returns the new message ID.
func (m *Messenger) SendTextReply(accountID uint64, chatID uint64, replyTo uint64, text string) (uint64, error) {
	msgID, err := m.rpc.SendMsg(dc.AccountId(accountID), dc.ChatId(chatID), dc.MsgData{
		Text:            text,
		QuotedMessageId: dc.MsgId(replyTo),
	})
	if err != nil {
		return 0, err
	}
	return uint64(msgID), nil
}

// SendMediaReply sends a media file as a quote-reply and returns the new message ID.
// mediaType must be a domain media type constant (image/sticker/gif/video).
func (m *Messenger) SendMediaReply(accountID uint64, chatID uint64, replyTo uint64, filePath string, mediaType string) (uint64, error) {
	viewType := mapMediaTypeToViewType(mediaType)
	if viewType == "" {
		return 0, fmt.Errorf("unknown media type: %s", mediaType)
	}

	msgID, err := m.rpc.SendMsg(dc.AccountId(accountID), dc.ChatId(chatID), dc.MsgData{
		File:            filePath,
		ViewType:        viewType,
		QuotedMessageId: dc.MsgId(replyTo),
	})
	if err != nil {
		return 0, err
	}
	return uint64(msgID), nil
}

// SendReaction sends a reaction emoji on a message.
func (m *Messenger) SendReaction(accountID uint64, msgID uint64, reaction string) error {
	_, err := m.rpc.SendReaction(dc.AccountId(accountID), dc.MsgId(msgID), reaction)
	return err
}

// SendTextMessage sends a plain text message (no quote-reply).
func (m *Messenger) SendTextMessage(accountID uint64, chatID uint64, text string) error {
	_, err := m.rpc.MiscSendTextMessage(dc.AccountId(accountID), dc.ChatId(chatID), text)
	return err
}

// DownloadMessage downloads the full media content of a message.
func (m *Messenger) DownloadMessage(accountID uint64, msgID uint64) error {
	return m.rpc.DownloadFullMessage(dc.AccountId(accountID), dc.MsgId(msgID))
}

// mapViewTypeToMediaType maps Delta Chat view types to domain media type constants.
func mapViewTypeToMediaType(viewType dc.MsgType) string {
	switch viewType {
	case dc.MsgImage:
		return domain.MediaTypeImage
	case dc.MsgSticker:
		return domain.MediaTypeSticker
	case dc.MsgGif:
		return domain.MediaTypeGIF
	case dc.MsgVideo:
		return domain.MediaTypeVideo
	default:
		return ""
	}
}

// mapMediaTypeToViewType maps domain media type constants back to Delta Chat view types.
func mapMediaTypeToViewType(mediaType string) dc.MsgType {
	switch mediaType {
	case domain.MediaTypeImage:
		return dc.MsgImage
	case domain.MediaTypeSticker:
		return dc.MsgSticker
	case domain.MediaTypeGIF:
		return dc.MsgGif
	case domain.MediaTypeVideo:
		return dc.MsgVideo
	default:
		return ""
	}
}

// mapDownloadState maps Delta Chat download state strings to domain constants.
func mapDownloadState(state dc.DownloadState) domain.DownloadState {
	switch state {
	case dc.DownloadDone:
		return domain.DownloadDone
	case dc.DownloadAvailable:
		return domain.DownloadAvailable
	default:
		return domain.DownloadState(state)
	}
}
