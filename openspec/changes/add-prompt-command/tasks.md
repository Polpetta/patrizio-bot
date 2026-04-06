## 1. Configuration

- [x] 1.1 Add OpenAI config keys to `internal/config/config.go`: `openai_base_url`, `openai_api_key`, `openai_model` (default: `gpt-4o-mini`), `openai_max_history` (default: `50`), `openai_allowed_chat_ids` (default: empty list), `openai_system_prompt` (default: `You are a helpful assistant.`). Add corresponding methods to the `Config` struct.
- [x] 1.2 Update the `domain.Config` interface in `internal/domain/ports.go` with new methods: `OpenAIBaseURL() string`, `OpenAIAPIKey() string`, `OpenAIModel() string`, `OpenAIMaxHistory() int`, `OpenAIAllowedChatIDs() []int64`, `OpenAISystemPrompt() string`.
- [x] 1.3 Add config tests for new keys (env vars, TOML, defaults).

## 2. Domain Models and Ports

- [x] 2.1 Add `ChatMessage` struct (Role, Content fields) and `AIClient` interface with `ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error)` to `internal/domain/ports.go`.
- [x] 2.2 Add `ConversationRepository` interface to `internal/domain/ports.go` with methods: `SaveMessage(ctx context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string) error`, `GetThreadChain(ctx context.Context, leafMsgID int64, limit int) ([]ChatMessage, error)`, `IsConversationMessage(ctx context.Context, msgID int64) (bool, *int64, error)` (returns whether the MsgId exists and the thread root ID).
- [x] 2.3 Add `CommandPrompt` constant to `internal/domain/command.go` and update `GetCommandType()` regex to recognize `/prompt`.
- [x] 2.4 Add `ParsePromptCommand(text string) (string, error)` function to `internal/domain/command.go` that extracts the message text after `/prompt `.
- [x] 2.5 Add unit tests for prompt command parsing and `GetCommandType` updates.
- [x] 2.6 Add `AIClient` and `ConversationRepository` fields to the `Dependencies` struct in `internal/domain/deps.go`.

## 3. Database Layer

- [x] 3.1 Create migration `003_conversations.up.sql` in `migrations/` with a `conversation_messages` table: `id INTEGER PRIMARY KEY`, `thread_root_id INTEGER NOT NULL` (MsgId of the original /prompt), `msg_id INTEGER NOT NULL UNIQUE` (Delta Chat MsgId), `parent_msg_id INTEGER` (MsgId of the quoted/parent message, NULL for root), `role TEXT NOT NULL`, `content TEXT NOT NULL`, `created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`. Add indexes on `(thread_root_id)` and `(msg_id)`.
- [x] 3.2 Create sqlc query file `queries/conversations.sql` with queries: insert message, check if msg_id exists (returning thread_root_id), get thread chain by walking parent pointers from a leaf msg_id up to root (recursive CTE) with limit, ordered chronologically.
- [x] 3.3 Run `make sqlc` to generate Go code in `internal/database/queries/`.
- [x] 3.4 Implement `ConversationRepository` in `internal/adapter/sqlite/conversation.go` using the generated sqlc queries.
- [x] 3.5 Add integration tests for `ConversationRepository` using an in-memory SQLite database with goose migrations applied (following the project's existing repository test pattern). Test: save messages, get chain, check existence, verify chain ordering and limit.

## 4. OpenAI Client Adapter

- [x] 4.1 Create `internal/adapter/openai/client.go` implementing the `domain.AIClient` interface. Use the official `github.com/openai/openai-go/v3` SDK: create a client with `openai.NewClient(option.WithAPIKey(...), option.WithBaseURL(...))`, call `client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{Messages: [...], Model: ...})`, and extract the response from `chatCompletion.Choices[0].Message.Content`. The adapter lives behind the `domain.AIClient` port interface, following the project's hex architecture.
- [x] 4.2 Add proper error handling: SDK errors, empty choices. Return user-friendly errors without leaking raw API details.
- [x] 4.3 Add unit tests for the OpenAI client adapter using `httptest.NewServer` as a fake API backend that the SDK client connects to (via `option.WithBaseURL`). Tests verify request structure, successful response parsing, and error handling.

## 5. Message Handler Updates

- [x] 5.1 Add `handlePromptCommand` function to `internal/bot/handler.go`: create a new thread (thread_root=own MsgId), build the messages array (prepend system prompt from config if non-empty, then user message), send to API via `AIClient.ChatCompletion`, save both user and assistant messages to ConversationRepository, send response as quote-reply. The assistant message's MsgId is the MsgId returned by `rpc.SendMsg`.
- [x] 5.2 Add `handleThreadContinuation` function to `internal/bot/handler.go`: given a message that quotes a Patrizio conversation message, reconstruct the chain via `ConversationRepository.GetThreadChain`, prepend system prompt from config if non-empty, append the new user message, call `AIClient.ChatCompletion`, save both messages, send response as quote-reply.
- [x] 5.3 Add `isThreadContinuation` helper that checks `msg.Quote`, calls `ConversationRepository.IsConversationMessage` for the quoted MsgId, and returns the thread context if it's a continuation.
- [x] 5.4 Update `handleGroupMessage` in `internal/bot/handler.go`: after command dispatch, before filter matching, check for thread continuation and dispatch to `handleThreadContinuation` if detected.
- [x] 5.5 Update `handleDMMessage` in `internal/bot/handler.go`: check for `/prompt` command first, then check for thread continuation, then fall back to help text.
- [x] 5.6 Update `helpText` constant to include `/prompt` command documentation and explain the reply-to-continue behavior.
- [x] 5.7 Handle the case where `AIClient` is nil (unconfigured): reply with a configuration error message for both `/prompt` and thread continuations.
- [x] 5.8 Add chat ID allowlist enforcement: before processing `/prompt` or a thread continuation, check if `OpenAIAllowedChatIDs()` is non-empty and if so, verify the current chat ID is in the list. Reply with a "not authorized" error if denied.
- [x] 5.9 Add `rpcClient` interface methods if any new RPC calls are needed (verify existing methods suffice — `GetMessage` and `SendMsg` should cover it).
- [x] 5.10 Add handler unit tests using mock implementations of port interfaces (AIClient, ConversationRepository, Config, rpcClient). Test `/prompt` (new thread creation, response delivery, API error, unconfigured) and thread continuation (valid continuation, non-conversation quote, no quote). Include tests for allowlist enforcement (allowed chat, denied chat, empty allowlist). Follow the project's existing handler test patterns in `internal/bot/handler_test.go`.

## 6. Bot Wiring

- [x] 6.1 Update `bot.BuildDependencies` in `internal/bot/bot.go` to create the OpenAI client adapter (if configured) and `ConversationRepository`, wiring them into the `Dependencies` struct.
- [x] 6.2 Update `cmd/patrizio/main.go` if any new initialization steps are needed (verify existing flow suffices).

## 7. Integration and Verification

- [x] 7.1 Run `make test` to ensure all existing and new tests pass.
- [x] 7.2 Run `make lint` to ensure no linting errors.
- [x] 7.3 Run `make build` to verify the binary compiles.
- [x] 7.4 Update `patrizio.toml` sample config with commented-out OpenAI configuration examples.
