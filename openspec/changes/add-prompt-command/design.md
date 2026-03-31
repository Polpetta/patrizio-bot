## Context

Patrizio is a Delta Chat bot that currently responds to messages exclusively through pre-configured keyword filters. The bot processes messages via `OnNewMsg` callbacks, routing group messages to command handlers or the filter matching engine, while DMs only return static help text. Configuration is managed through Viper (TOML + `PATRIZIO_*` env vars). The codebase follows a ports-and-adapters architecture with domain interfaces in `internal/domain/ports.go` and concrete implementations in `internal/adapter/`.

Users want conversational AI capabilities within Delta Chat. The `/prompt` command will integrate an OpenAI-compatible chat completion API, with conversation context maintained through Delta Chat's native reply-chain mechanism — users reply to Patrizio's messages to continue a thread, and each reply chain forms an independent conversation.

## Goals / Non-Goals

**Goals:**
- Add `/prompt <message>` command accessible in both DMs and group chats
- Maintain conversation context via reply chains: replying to a Patrizio conversation message continues the thread without needing `/prompt` again
- Support multiple independent conversation threads per chat (each `/prompt` starts a new one)
- Support any OpenAI-compatible API endpoint (OpenAI, Ollama, LMStudio, etc.)
- Follow the existing architectural patterns (ports/adapters, sqlc, goose migrations)
- Make the feature cleanly optional — if no OpenAI config is set, `/prompt` replies with a configuration error
- Restrict `/prompt` usage via a configurable chat ID allowlist to prevent abuse

**Non-Goals:**
- Streaming responses (the bot will wait for the full completion before replying)
- System prompt customization via chat commands (system prompt is a static config value)
- Token counting, rate limiting, or cost management
- Support for function calling, tool use, or other advanced OpenAI features
- Image generation or vision capabilities

## Decisions

### 1. Use the official OpenAI Go SDK (`openai-go` v3)

**Decision:** Use `github.com/openai/openai-go/v3` to call the chat completion API. The client is created with `openai.NewClient(option.WithAPIKey(...), option.WithBaseURL(...))` and completions are requested via `client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{Messages: [...], Model: ...})`. The response is accessed via `chatCompletion.Choices[0].Message.Content`.

**Rationale:** The official SDK handles authentication headers, request serialization, response deserialization, and error mapping out of the box. It supports custom base URLs via `option.WithBaseURL(...)`, making it compatible with OpenAI, Ollama, LMStudio, and other providers. Using the SDK eliminates boilerplate HTTP code and reduces the surface area for bugs. The concrete implementation lives in `internal/adapter/openai/` behind the `domain.AIClient` port interface, consistent with the project's hex architecture.

**Alternatives considered:**
- `net/http` directly — Fewer dependencies but requires manual JSON serialization/deserialization, header management, and error parsing. More code to maintain for no meaningful benefit.
- `sashabaranov/go-openai` (community SDK) — Third-party, less guaranteed long-term support compared to the official SDK.

### 2. Reply-chain threading instead of per-chat history

**Decision:** Conversation context is scoped to a reply chain, not a chat ID. A `/prompt` command starts a new thread. Patrizio's response quotes the user's message. If anyone replies to Patrizio's response (or to any subsequent Patrizio message in that chain), the reply is treated as a thread continuation — no `/prompt` prefix required. The full chain from the original `/prompt` down to the current message is assembled and sent as context to the API.

**Rationale:** This maps naturally to how Delta Chat conversations work — users reply to messages to continue a topic. It allows multiple independent AI conversations in the same group chat. It also means users don't need to remember a command prefix for follow-ups; they just reply to Patrizio's last message. Starting a fresh conversation is trivial: send a new `/prompt`.

**Alternatives considered:**
- Per-chat-ID history — Simpler to implement but mixes unrelated conversations in the same chat. No way to have parallel threads. Requires an explicit `/reset` command to start fresh.

### 3. Persist conversation messages in SQLite with Delta Chat message IDs

**Decision:** Store each conversation message in a SQLite table that records: a thread ID (the Delta Chat MsgId of the root `/prompt` message), the Delta Chat MsgId of this message, the MsgId of the parent message (the message being replied to), the role (`user` or `assistant`), the content, and a timestamp. Use sqlc-generated queries consistent with the existing pattern.

**Rationale:** The thread is reconstructed by walking from the current message up through parent pointers to the root. Storing Delta Chat MsgIds is essential because the bot needs to check whether a quoted message belongs to an active conversation thread when a non-command message arrives. SQLite persistence ensures threads survive bot restarts.

**Alternatives considered:**
- In-memory map — Lost on restart. Unacceptable.
- Storing only the root ID and a flat ordered list — Harder to reconstruct the exact chain if the reply tree branches (e.g., two users reply to the same Patrizio message, creating two sub-threads).

### 4. Define an `AIClient` port interface in the domain layer

**Decision:** Create a `domain.AIClient` interface with a single method: `ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error)`. The concrete implementation lives in `internal/adapter/openai/`.

**Rationale:** Follows the existing ports-and-adapters pattern (`FilterRepository`, `MediaStorage`, `Config`). Enables testing the handler with a mock AI client. Keeps the domain layer free of HTTP/API concerns.

### 5. Extend DM handler to support commands and thread continuations

**Decision:** Modify the DM handler to check for the `/prompt` command and for reply-chain continuations before falling back to help text.

**Rationale:** Currently DMs always return help text. For `/prompt` to work in DMs, the handler needs to recognize commands and detect when a reply targets a Patrizio conversation message. This is a minimal change — check command type first, then check for thread continuation, then fall through to help text.

### 6. No `/reset` command — new `/prompt` starts a fresh thread

**Decision:** There is no `/reset` command. Each `/prompt` creates an independent conversation thread. To "start fresh," users simply send a new `/prompt` instead of replying to an existing chain.

**Rationale:** With reply-chain threading, threads are naturally independent. A `/reset` that clears "all history" doesn't make sense when there are multiple concurrent threads. Users control context implicitly by choosing whether to reply to an existing thread or start a new one.

### 7. Configurable history depth with a sensible default

**Decision:** Add an `openai_max_history` config key (default: 50 messages) that caps the number of messages sent as context per thread. When the chain exceeds this limit, only the most recent N messages are sent to the API. The full chain remains in the database.

**Rationale:** OpenAI APIs have token limits. Very long reply chains would eventually exceed them. A configurable cap lets operators tune for their model's context window while providing a reasonable default.

### 8. Thread continuation detection

**Decision:** When a non-command message arrives and it quotes another message (`msg.Quote != nil`), the handler checks if the quoted MsgId exists in the conversation messages table. If it does, the message is a thread continuation. If not, it falls through to normal processing (filter matching in groups, help text in DMs).

**Rationale:** This is the minimal check needed. The bot only stores MsgIds for messages it participates in (user prompts + assistant responses), so the lookup is precise. False positives are impossible — a message can only continue a thread if it explicitly replies to a Patrizio conversation message.

### 9. Chat ID allowlist for abuse prevention

**Decision:** Add an `openai_allowed_chat_ids` config key that accepts a list of Delta Chat chat IDs. When the list is non-empty, only `/prompt` commands and thread continuations originating from those chat IDs are processed — all others receive a "not authorized" error reply. When the list is empty or unset, all chats are allowed (open access). The check operates at the chat level: if a group chat ID is in the allowlist, all members of that group can use `/prompt`.

**Rationale:** Since the bot operator pays for API calls (or hosts the model), they need a way to control who can use the feature. A chat-level allowlist is the simplest effective control — it avoids needing to manage individual user IDs and aligns with how Delta Chat groups work (if you're in the group, you're trusted). The default of "allow all" keeps the setup frictionless for personal or small deployments.

**Alternatives considered:**
- Per-user allowlist — More granular but harder to manage. Delta Chat contact IDs are less convenient than chat IDs (which the bot operator can easily obtain). Not worth the complexity for the initial implementation.
- No access control — Simpler, but any user who adds the bot to a group could rack up API costs. Unacceptable for public-facing deployments.

### 10. Configurable system prompt with a sensible default

**Decision:** Add an `openai_system_prompt` config key (default: `You are a helpful assistant.`). When non-empty, the system prompt is prepended as a system role message to every chat completion request, before the conversation history. The system prompt is not persisted in the conversation messages table — it is injected at request time from the current config value.

**Rationale:** A system prompt gives the model personality and instructions. Without one, the model uses raw provider defaults, which vary across providers and may produce inconsistent behavior. A static config value keeps it simple while allowing operators to customize the bot's behavior (e.g., "You are Patrizio, a friendly assistant in a Delta Chat group. Be concise."). The default of "You are a helpful assistant." matches the common baseline and works across all providers.

## Risks / Trade-offs

- **[API latency]** → OpenAI API calls may take seconds. The bot's `OnNewMsg` callback will block during this time. Since Delta Chat bots process messages sequentially per account, this won't miss messages but may delay filter responses. **Mitigation:** Accept for now; async processing is a future optimization.

- **[Token costs]** → No built-in cost controls. **Mitigation:** The `openai_max_history` cap limits context size. The chat ID allowlist restricts who can trigger API calls. Operators can also use local models (Ollama) to avoid costs entirely.

- **[API key exposure]** → The API key is stored in config (TOML or env var). **Mitigation:** Recommend env vars over TOML for secrets. The config file should not be committed to version control (already in `.gitignore` patterns).

- **[Provider compatibility]** → Not all OpenAI-compatible providers implement the API identically. **Mitigation:** Use only the core chat completion fields (role, content) that all providers support. The official SDK's `option.WithBaseURL(...)` is designed for provider flexibility. Avoid provider-specific extensions.

- **[Database growth]** → Conversation thread messages accumulate in SQLite. **Mitigation:** Threads are naturally bounded by how long users continue replying. A future enhancement could prune old threads after a configurable TTL.

- **[Reply chain resolution cost]** → Walking the chain from leaf to root requires multiple DB lookups or a recursive query. **Mitigation:** Conversation chains are practically short (tens of messages). A single recursive CTE query can assemble the chain efficiently.

- **[Branching threads]** → Two users might reply to the same Patrizio message, creating a fork. Each branch is an independent sub-chain from that point forward. **Mitigation:** This is handled naturally — each new reply creates its own path to the root. Both branches share the common prefix but diverge at the fork point.
