## 1. Extend Logger Interface

- [x] 1.1 Add `Warnf(string, ...interface{})` to the `handlerLogger` interface in `internal/bot/handler.go`

## 2. Extract processMessage Function

- [x] 2.1 Extract the goroutine body (lines 87–119) from `newMsgHandler` into a new `processMessage(rpc rpcClient, logger handlerLogger, accID deltachat.AccountId, msgID deltachat.MsgId, deps *domain.Dependencies)` function
- [x] 2.2 Simplify `newMsgHandler`'s goroutine to `go processMessage(bot.Rpc, cli.GetLogger(accID), accID, msgID, deps)`

## 3. Add Unit Tests for processMessage

- [x] 3.1 Add `TestProcessMessage_GetMessageError` — verifies `Errorf` is logged when `GetMessage` fails, nothing sent
- [x] 3.2 Add `TestProcessMessage_IgnoresSpecialContact` — verifies no action when `FromId <= ContactLastSpecial`
- [x] 3.3 Add `TestProcessMessage_GetChatInfoError` — verifies `Errorf` is logged when `GetBasicChatInfo` fails, nothing sent
- [x] 3.4 Add `TestProcessMessage_RoutesGroupChat` — verifies group handler is reached (filter repo called or RPC send triggered)
- [x] 3.5 Add `TestProcessMessage_RoutesSingleChat` — verifies DM handler is reached (help text sent via `MiscSendTextMessage`)
- [x] 3.6 Add `TestProcessMessage_UnknownChatTypeWarns` — verifies `Warnf` is logged and nothing is sent

## 4. Verify

- [x] 4.1 Run `go test ./internal/bot/...` and confirm all tests pass
