## Context

`newMsgHandler` (in `internal/bot/handler.go`) returns a `deltachat.NewMsgHandler` closure. After adding goroutine-per-message support, the body of that goroutine contains all the routing logic: fetching the message, filtering special contacts, fetching chat info, and dispatching to `handleGroupMessage` or `handleDMMessage`. This logic is entirely untested because:

1. It is inside a goroutine — tests would need synchronization to observe results.
2. It depends on `*botcli.BotCli` to obtain a logger — tests cannot construct a real `BotCli`.

All inner handler functions (`handleGroupMessage`, `handleDMMessage`, `handleFilterCommand`, etc.) are already well-tested by calling them directly. The gap is exclusively the routing glue in `newMsgHandler`.

## Goals / Non-Goals

**Goals:**
- Make the routing logic inside `newMsgHandler`'s goroutine callable synchronously from tests.
- Cover the six routing paths with unit tests: GetMessage error, special-contact filtering, GetBasicChatInfo error, group dispatch, DM dispatch, unknown chat type warning.
- Keep the change surgical — no refactoring of inner handlers or test infrastructure beyond what is strictly necessary.

**Non-Goals:**
- Testing `newMsgHandler` itself (it remains a thin goroutine launcher; no meaningful logic to assert on).
- Integration testing with a real `deltachat.Bot` or `botcli.BotCli`.
- Changing any handler behavior.

## Decisions

**Extract a `processMessage` function.**
Move lines 87–119 of the goroutine body into a standalone function:
```go
func processMessage(
    rpc rpcClient,
    logger handlerLogger,
    accID deltachat.AccountId,
    msgID deltachat.MsgId,
    deps *domain.Dependencies,
)
```
`newMsgHandler` becomes:
```go
return func(bot *deltachat.Bot, accID deltachat.AccountId, msgID deltachat.MsgId) {
    go processMessage(bot.Rpc, cli.GetLogger(accID), accID, msgID, deps)
}
```
This is the minimal extraction needed: no new types, no new files.

**Add `Warnf` to `handlerLogger`.**
The routing body already calls `logger.Warnf(...)` for unknown chat types (line 118). The existing `handlerLogger` interface only declares `Infof` and `Errorf`. Adding `Warnf` is the smallest change that makes the interface complete and lets `mockLogger` (which already has `Warnf`) satisfy it without modification.

**No synchronization mechanism in `newMsgHandler`.**
Tests call `processMessage` directly (synchronously). The goroutine wrapper in production code is left as-is. This avoids adding WaitGroup or channel fields that would complicate production code.

## Risks / Trade-offs

- **`Warnf` on `handlerLogger`**: Any external code implementing `handlerLogger` (none in this repo today, since it is unexported) would need to add `Warnf`. Risk is negligible.
- **Goroutine untested**: The one-liner `go processMessage(...)` inside `newMsgHandler` remains untested. This is acceptable — it is too trivial to warrant the complexity of an integration test.
