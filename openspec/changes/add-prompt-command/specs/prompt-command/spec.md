## ADDED Requirements

### Requirement: Prompt command parsing
The system SHALL recognize `/prompt <message>` as a valid command. The `<message>` portion is everything after `/prompt ` (the command prefix plus one space). The message MUST NOT be empty.

#### Scenario: Valid prompt command
- **WHEN** a user sends `/prompt What is the capital of France?`
- **THEN** the system parses the command and extracts `What is the capital of France?` as the message

#### Scenario: Empty prompt rejected
- **WHEN** a user sends `/prompt` or `/prompt   ` (only whitespace after command)
- **THEN** the system replies with an error message indicating that a message is required

### Requirement: Chat completion via OpenAI-compatible API
The system SHALL send the user's message along with the conversation thread history to a configurable OpenAI-compatible chat completion endpoint. When `openai_system_prompt` is configured (non-empty), the system SHALL prepend it as a system role message before the conversation messages in every API request. The request SHALL include the `model` from configuration and the conversation messages array with `role` and `content` fields.

#### Scenario: Successful API call
- **WHEN** a `/prompt` command or thread continuation is received and the OpenAI endpoint is configured
- **THEN** the system sends a POST request to `{openai_base_url}/chat/completions` with the thread's message history and returns the assistant's response as a reply message (the base URL is expected to include the `/v1` path segment, e.g. `https://api.openai.com/v1`)

#### Scenario: API error handling
- **WHEN** the OpenAI API returns an error (HTTP status >= 400) or the request fails
- **THEN** the system replies with an error message to the user indicating the request failed, without exposing raw API error details

#### Scenario: Missing configuration
- **WHEN** a `/prompt` command is received but `openai_base_url` or `openai_api_key` is not configured
- **THEN** the system replies with an error message indicating that the OpenAI integration is not configured

### Requirement: Reply-chain conversation threading
The system SHALL maintain conversation context through Delta Chat reply chains. A `/prompt` command starts a new conversation thread. When Patrizio responds, it quotes the user's message. If any user replies to a Patrizio message that belongs to a conversation thread, the system SHALL treat that reply as a continuation of the thread — no `/prompt` prefix is required. The system SHALL assemble the full message chain from the original `/prompt` to the current message and send it as context to the API.

#### Scenario: New thread started with /prompt
- **WHEN** a user sends `/prompt Hello` in any chat
- **THEN** the system creates a new conversation thread, sends `Hello` (role: `user`) to the API, stores both the user message and the assistant's response with the thread root set to the `/prompt` message's MsgId, and replies by quoting the user's message

#### Scenario: Thread continuation via reply
- **WHEN** User A receives Patrizio's response and User B replies to that response with `Tell me more`
- **THEN** the system detects the reply targets a Patrizio conversation message, reconstructs the chain (User A's original prompt, Patrizio's response, User B's `Tell me more`), sends the full chain to the API, stores the new messages, and replies by quoting User B's message

#### Scenario: Deep reply chain
- **WHEN** a conversation has gone through multiple back-and-forth exchanges and a user replies to Patrizio's latest message
- **THEN** the system reconstructs the entire chain from root to current message and sends it as context

#### Scenario: Multiple independent threads in same chat
- **WHEN** User A starts `/prompt Topic A` and User B starts `/prompt Topic B` in the same group
- **THEN** each thread is tracked independently, and replies to each Patrizio response only include context from their respective thread

#### Scenario: Branching thread
- **WHEN** two users each reply to the same Patrizio message in a thread
- **THEN** each reply creates an independent sub-chain from that point forward, sharing the common prefix but diverging at the fork

### Requirement: Conversation message persistence
The system SHALL persist each conversation message in a SQLite table with: a thread root ID (the Delta Chat MsgId of the original `/prompt` message), the Delta Chat MsgId of this message, the MsgId of the parent message (the message being replied to), the role (`user` or `assistant`), the message content, and a creation timestamp. This table is used for both thread reconstruction and continuation detection.

#### Scenario: Storing a /prompt exchange
- **WHEN** a user sends `/prompt Hello` and the API responds with `Hi there`
- **THEN** the system stores two rows: one for the user message (role: `user`, content: `Hello`, thread_root=own MsgId, parent=NULL) and one for the assistant response (role: `assistant`, content: `Hi there`, thread_root=root MsgId, parent=user's MsgId)

#### Scenario: Storing a continuation exchange
- **WHEN** a user replies `Thanks` to a Patrizio conversation message and the API responds with `You're welcome`
- **THEN** the system stores two rows: one for the user message (role: `user`, content: `Thanks`, parent=quoted Patrizio MsgId) and one for the assistant response (role: `assistant`, content: `You're welcome`, parent=user's MsgId), both sharing the thread root of the original `/prompt`

#### Scenario: History survives bot restart
- **WHEN** the bot restarts and a user replies to a Patrizio conversation message from a prior session
- **THEN** the system loads the persisted thread from SQLite and includes the full chain in the API request

### Requirement: Thread continuation detection
The system SHALL detect conversation thread continuations by checking whether an incoming non-command message quotes a MsgId that exists in the conversation messages table. If the quoted MsgId is found, the message is a thread continuation. If not, it falls through to normal processing.

#### Scenario: Reply to Patrizio conversation message
- **WHEN** a non-command message arrives that quotes a MsgId present in the conversation messages table
- **THEN** the system treats it as a thread continuation and dispatches to the prompt handler

#### Scenario: Reply to non-conversation message
- **WHEN** a non-command message arrives that quotes a MsgId NOT in the conversation messages table
- **THEN** the system ignores it for conversation purposes and processes it normally (filter matching in groups, help text in DMs)

#### Scenario: Message with no quote
- **WHEN** a non-command message arrives with no quoted message
- **THEN** the system processes it normally (filter matching in groups, help text in DMs)

### Requirement: Configurable history depth
The system SHALL limit the number of messages sent to the API per thread based on the `openai_max_history` configuration value (default: 50). When the chain exceeds this limit, only the most recent N messages SHALL be sent. The full chain SHALL remain in the database.

#### Scenario: Chain within limit
- **WHEN** a thread has 10 messages and `openai_max_history` is 50
- **THEN** all 10 messages are sent to the API

#### Scenario: Chain exceeds limit
- **WHEN** a thread has 80 messages in the chain and `openai_max_history` is 50
- **THEN** only the 50 most recent messages in the chain are sent to the API

### Requirement: Response delivery
The system SHALL send the assistant's response as a quote-reply to the user's message (whether it was a `/prompt` command or a thread continuation reply), consistent with how filter text responses are delivered.

#### Scenario: Response sent as quote-reply to /prompt
- **WHEN** the API returns a successful response to a `/prompt` command
- **THEN** the bot sends the assistant's response text as a message that quotes the original `/prompt` message

#### Scenario: Response sent as quote-reply to continuation
- **WHEN** the API returns a successful response to a thread continuation
- **THEN** the bot sends the assistant's response text as a message that quotes the user's continuation message

### Requirement: AIClient port interface
The system SHALL define an `AIClient` interface in the domain layer with a method `ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error)` where `ChatMessage` has `Role` and `Content` fields. The concrete implementation SHALL use the official `github.com/openai/openai-go/v3` SDK to call the OpenAI-compatible chat completion endpoint. The implementation SHALL live in `internal/adapter/openai/` behind the domain port interface, following the project's hexagonal architecture.

#### Scenario: Interface used for dependency injection
- **WHEN** the bot is initialized
- **THEN** the `AIClient` implementation is created with the configured base URL, API key, and model, and injected into the dependencies struct

#### Scenario: Testability via mock
- **WHEN** handler tests need to verify prompt command behavior
- **THEN** the `AIClient` interface can be mocked without making real HTTP calls

### Requirement: Chat ID allowlist
The system SHALL enforce a chat ID allowlist for `/prompt` commands and thread continuations. When `openai_allowed_chat_ids` is configured with a non-empty list, only messages from those chat IDs SHALL be processed. Messages from non-allowed chats SHALL receive a "not authorized" error reply. When the list is empty or unset, all chats SHALL be allowed. The allowlist operates at the chat level: if a group chat ID is in the allowlist, all members of that group are permitted to use `/prompt` and continue threads.

#### Scenario: Chat in allowlist
- **WHEN** a `/prompt` command or thread continuation arrives from a chat whose ID is in `openai_allowed_chat_ids`
- **THEN** the system processes it normally

#### Scenario: Chat not in allowlist
- **WHEN** a `/prompt` command or thread continuation arrives from a chat whose ID is NOT in `openai_allowed_chat_ids` and the list is non-empty
- **THEN** the system replies with an error message indicating the chat is not authorized to use this feature

#### Scenario: Empty allowlist allows all
- **WHEN** `openai_allowed_chat_ids` is empty or not configured
- **THEN** all chats are allowed to use `/prompt` and thread continuations

#### Scenario: Group member access
- **WHEN** a group chat ID is in the allowlist and any member of that group sends `/prompt` or continues a thread
- **THEN** the system processes the request — authorization is per chat, not per user

### Requirement: Hexagonal architecture and testing patterns
All new code SHALL follow the project's existing hexagonal (ports-and-adapters) architecture. Port interfaces SHALL be defined in `internal/domain/ports.go`. Concrete adapters SHALL live in `internal/adapter/`. Handler tests SHALL use mock implementations of port interfaces (AIClient, ConversationRepository, Config) to verify behavior without external dependencies. Repository integration tests SHALL use an in-memory SQLite database with goose migrations applied. The OpenAI adapter SHALL be tested using the `openai-go` SDK's test utilities or an HTTP test server that the SDK client connects to.
