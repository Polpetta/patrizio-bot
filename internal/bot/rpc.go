package bot

import "github.com/chatmail/rpc-client-go/deltachat"

// rpcClient defines the subset of deltachat.Rpc methods used by the handlers.
// The real *deltachat.Rpc satisfies this interface implicitly.
// Tests can supply a mock implementation.
type rpcClient interface {
	GetMessage(accountID deltachat.AccountId, msgID deltachat.MsgId) (*deltachat.MsgSnapshot, error)
	GetBasicChatInfo(accountID deltachat.AccountId, chatID deltachat.ChatId) (*deltachat.BasicChatSnapshot, error)
	MiscSendTextMessage(accountID deltachat.AccountId, chatID deltachat.ChatId, text string) (deltachat.MsgId, error)
	SendReaction(accountID deltachat.AccountId, msgID deltachat.MsgId, reaction ...string) (deltachat.MsgId, error)
	DownloadFullMessage(accountID deltachat.AccountId, msgID deltachat.MsgId) error
}
