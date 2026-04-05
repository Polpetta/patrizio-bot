## Why

The `newMsgHandler` function now spawns a goroutine per incoming message, but the routing logic inside that goroutine (special-contact filtering, chat-type dispatch, RPC error handling) has zero test coverage. The goroutine body is inlined and coupled to `*botcli.BotCli`, making it impossible to call synchronously from tests.

## What Changes

- Add `Warnf` to the `handlerLogger` interface so the routing layer can log unknown chat types through the same injectable logger used by all inner handlers.
- Extract the goroutine body from `newMsgHandler` into a new `processMessage` function with a testable synchronous signature (`rpc rpcClient`, `logger handlerLogger`, `accID`, `msgID`, `deps`).
- Simplify `newMsgHandler` to a single `go processMessage(...)` call.
- Add unit tests for `processMessage` covering: special-contact filtering, group routing, DM routing, unknown chat type warning, and RPC error paths.

## Capabilities

### New Capabilities

- `message-routing-testability`: Unit-testable entry-point (`processMessage`) for the full per-message routing pipeline.

### Modified Capabilities

- `message-handling`: The routing logic already described in the spec is now fully exercised by tests; no requirement changes, the implementation is refactored to enable testing.

## Impact

- **`internal/bot/handler.go`**: `handlerLogger` gains `Warnf`; goroutine body extracted to `processMessage`.
- **`internal/bot/handler_test.go`**: `mockLogger` already implements `Warnf`; ~6 new test functions added for `processMessage`.
- No public API, database schema, or dependency changes.
