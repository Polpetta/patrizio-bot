// Package deltachat provides a Messenger adapter backed by the Delta Chat RPC client.
package deltachat

import (
	"fmt"

	dc "github.com/chatmail/rpc-client-go/v2/deltachat"

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
func (m *Messenger) FetchMessage(accountID uint32, msgID uint32) (*domain.IncomingMessage, error) {
	msg, err := m.rpc.GetMessage(accountID, msgID)
	if err != nil {
		return nil, err
	}

	var filePath string
	if msg.File != nil {
		filePath = *msg.File
	}

	result := &domain.IncomingMessage{
		ID:            msg.Id,
		ChatID:        msg.ChatId,
		FromID:        msg.FromId,
		Text:          msg.Text,
		File:          filePath,
		MediaType:     mapViewTypeToMediaType(msg.ViewType),
		DownloadState: mapDownloadState(msg.DownloadState),
	}

	if msg.Quote != nil {
		if quote, ok := (*msg.Quote).(*dc.MessageQuoteWithMessage); ok {
			result.Quote = &domain.QuotedMessage{
				MessageID: quote.MessageId,
			}
		}
	}

	return result, nil
}

// FetchChatType retrieves the chat type for a given chat.
func (m *Messenger) FetchChatType(accountID uint32, chatID uint32) (domain.ChatType, error) {
	info, err := m.rpc.GetBasicChatInfo(accountID, chatID)
	if err != nil {
		return "", err
	}

	switch info.ChatType {
	case dc.ChatTypeGroup, dc.ChatTypeOutBroadcast, dc.ChatTypeInBroadcast, dc.ChatTypeMailinglist:
		return domain.ChatTypeGroup, nil
	case dc.ChatTypeSingle:
		return domain.ChatTypeSingle, nil
	default:
		return domain.ChatTypeUnknown, nil
	}
}

// SendTextReply sends a text message as a quote-reply and returns the new message ID.
func (m *Messenger) SendTextReply(accountID uint32, chatID uint32, replyTo uint32, text string) (uint32, error) {
	msgID, err := m.rpc.SendMsg(accountID, chatID, dc.MessageData{
		Text:            &text,
		QuotedMessageId: &replyTo,
	})
	if err != nil {
		return 0, err
	}
	return msgID, nil
}

// SendMediaReply sends a media file as a quote-reply and returns the new message ID.
// mediaType must be a domain media type constant (image/sticker/gif/video).
func (m *Messenger) SendMediaReply(accountID uint32, chatID uint32, replyTo uint32, filePath string, mediaType string) (uint32, error) {
	viewType := mapMediaTypeToViewType(mediaType)
	if viewType == "" {
		return 0, fmt.Errorf("unknown media type: %s", mediaType)
	}

	msgID, err := m.rpc.SendMsg(accountID, chatID, dc.MessageData{
		File:            &filePath,
		Viewtype:        &viewType,
		QuotedMessageId: &replyTo,
	})
	if err != nil {
		return 0, err
	}
	return msgID, nil
}

// SendReaction sends a reaction emoji on a message.
func (m *Messenger) SendReaction(accountID uint32, msgID uint32, reaction string) error {
	_, err := m.rpc.SendReaction(accountID, msgID, []string{reaction})
	return err
}

// SendTextMessage sends a plain text message (no quote-reply).
func (m *Messenger) SendTextMessage(accountID uint32, chatID uint32, text string) error {
	_, err := m.rpc.MiscSendTextMessage(accountID, chatID, text)
	return err
}

// DownloadMessage downloads the full media content of a message.
func (m *Messenger) DownloadMessage(accountID uint32, msgID uint32) error {
	return m.rpc.DownloadFullMessage(accountID, msgID)
}

// IsSpecialContact reports whether the given contact ID is a system/device contact.
func (m *Messenger) IsSpecialContact(fromID uint32) bool {
	return fromID <= dc.ContactLastSpecial
}

// FetchContactDisplayName retrieves the display name for a contact.
// Falls back to the contact's name, then email address if display name is empty.
func (m *Messenger) FetchContactDisplayName(accountID uint32, contactID uint32) (string, error) {
	contact, err := m.rpc.GetContact(accountID, contactID)
	if err != nil {
		return "", err
	}
	if contact.DisplayName != "" {
		return contact.DisplayName, nil
	}
	if contact.Name != "" {
		return contact.Name, nil
	}
	return contact.Address, nil
}

// mapViewTypeToMediaType maps Delta Chat view types to domain media type constants.
func mapViewTypeToMediaType(viewType dc.Viewtype) string {
	switch viewType {
	case dc.ViewtypeImage:
		return domain.MediaTypeImage
	case dc.ViewtypeSticker:
		return domain.MediaTypeSticker
	case dc.ViewtypeGif:
		return domain.MediaTypeGIF
	case dc.ViewtypeVideo:
		return domain.MediaTypeVideo
	default:
		return ""
	}
}

// mapMediaTypeToViewType maps domain media type constants back to Delta Chat view types.
func mapMediaTypeToViewType(mediaType string) dc.Viewtype {
	switch mediaType {
	case domain.MediaTypeImage:
		return dc.ViewtypeImage
	case domain.MediaTypeSticker:
		return dc.ViewtypeSticker
	case domain.MediaTypeGIF:
		return dc.ViewtypeGif
	case domain.MediaTypeVideo:
		return dc.ViewtypeVideo
	default:
		return ""
	}
}

// mapDownloadState maps Delta Chat download state strings to domain constants.
func mapDownloadState(state dc.DownloadState) domain.DownloadState {
	switch state {
	case dc.DownloadStateDone:
		return domain.DownloadDone
	case dc.DownloadStateAvailable:
		return domain.DownloadAvailable
	default:
		return domain.DownloadState(state)
	}
}
