## MODIFIED Requirements

### Requirement: Group message routing
The system SHALL identify group chat messages and route them to the appropriate handler. The system SHALL check for commands first (including `/prompt`). If the message is not a command, the system SHALL check whether it is a conversation thread continuation (i.e., it quotes a Patrizio conversation message). If it is a continuation, it SHALL dispatch to the prompt handler. Otherwise, it SHALL normalize the message and check for filter matches.

#### Scenario: Group message received
- **WHEN** a message arrives in a group chat from a regular user
- **THEN** the system checks for commands first, then checks for thread continuation, then falls through to filter matching if neither applies

#### Scenario: Prompt command in group
- **WHEN** a user sends `/prompt <message>` in a group chat
- **THEN** the system dispatches to the prompt command handler

#### Scenario: Thread continuation in group
- **WHEN** a user replies to a Patrizio conversation message in a group chat (without a command prefix)
- **THEN** the system detects the thread continuation and dispatches to the prompt handler with the reply chain context

#### Scenario: Non-conversation reply in group
- **WHEN** a user replies to a message that is NOT a Patrizio conversation message
- **THEN** the system falls through to filter matching as normal

### Requirement: DM help response
The system SHALL check DM messages for recognized commands (`/prompt`) and for conversation thread continuations before falling back to the static help response. If the message is a recognized command or a thread continuation, it SHALL be dispatched to the appropriate handler. Otherwise, the bot SHALL reply with the static help/usage text.

#### Scenario: User sends a DM to the bot
- **WHEN** a regular user sends a non-command, non-continuation message directly to the bot
- **THEN** the bot replies with the static help/usage text message

#### Scenario: User sends /prompt in DM
- **WHEN** a regular user sends `/prompt <message>` directly to the bot
- **THEN** the bot dispatches to the prompt command handler (not the help text)

#### Scenario: User continues thread in DM
- **WHEN** a user replies to a Patrizio conversation message in a DM
- **THEN** the bot dispatches to the prompt handler with the reply chain context

#### Scenario: Help text content
- **WHEN** the bot sends a DM help response
- **THEN** the message SHALL include the bot name, a brief description of its purpose, instructions to add it to a group, and documentation of the `/prompt` command
