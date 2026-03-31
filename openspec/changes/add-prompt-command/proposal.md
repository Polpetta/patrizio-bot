## Why

Patrizio currently only responds to messages via pre-configured filters. Users have no way to have interactive, context-aware conversations with the bot. Adding an `/prompt` command backed by an OpenAI-compatible API enables free-form conversational AI within Delta Chat, expanding the bot's utility from a static responder to an interactive assistant — available in both DMs and groups.

## What Changes

- Add a new `/prompt <message>` command that sends user messages to a configurable OpenAI-compatible chat completion endpoint and returns the response.
- Maintain conversation context via reply chains: a `/prompt` starts a new thread, and any reply to Patrizio's response in that thread continues the conversation. The full reply chain from the original `/prompt` is sent as context to the API. Multiple independent threads can coexist in the same chat.
- Add new configuration keys for the OpenAI endpoint: `openai_base_url`, `openai_api_key`, `openai_model`, and `openai_system_prompt`.
- Add a chat ID allowlist (`openai_allowed_chat_ids`) to restrict which chats can use `/prompt`. When empty or unset, all chats are allowed.
- Make `/prompt` available in both DMs and group chats (subject to allowlist).
- Detect conversation continuations: when a non-command message quotes a Patrizio message that belongs to a conversation thread, treat it as a follow-up (no `/prompt` prefix needed).
- Update help text to document the new `/prompt` command.

## Capabilities

### New Capabilities
- `prompt-command`: Handles parsing, dispatching, and responding to `/prompt <message>` commands, including reply-chain-based conversation threading and OpenAI API integration.

### Modified Capabilities
- `message-handling`: The `/prompt` command must be recognized as a valid command type and routed in both group and DM message handlers. Non-command messages that quote a Patrizio conversation message must be detected as thread continuations. DMs currently only show help text and need to support commands and thread continuations too.
- `bot-skeleton`: Config must be extended with OpenAI endpoint settings (`openai_base_url`, `openai_api_key`, `openai_model`, `openai_system_prompt`), and the new dependency (OpenAI client) must be wired into the bot lifecycle.

## Impact

- **Config**: New keys added (`openai_base_url`, `openai_api_key`, `openai_model`, `openai_system_prompt`, `openai_allowed_chat_ids`) with corresponding env vars (`PATRIZIO_OPENAI_BASE_URL`, etc.).
- **Domain**: New command type constant, new port interface for the AI client, new conversation thread model.
- **Database**: New table(s) to persist conversation messages with Delta Chat message IDs for reply-chain resolution.
- **Dependencies**: Official OpenAI Go SDK (`github.com/openai/openai-go/v3`) for calling the chat completion API.
- **Handler**: Message routing updated to recognize `/prompt` in both group and DM contexts, plus detection of reply-chain continuations for non-command messages.
- **DM behavior change**: DMs currently always reply with static help text; they will now also process `/prompt` commands and reply-chain continuations before falling back to help.
