# Message Handling

## Purpose

Defines how the bot receives, filters, routes, and responds to incoming messages from Delta Chat.

## Requirements

### Requirement: Message reception via OnNewMsg
The system SHALL register a `bot.OnNewMsg` callback during `OnBotInit` that receives all incoming messages across all accounts.

#### Scenario: Incoming message triggers handler
- **WHEN** the bot receives a new message from any chat
- **THEN** the `OnNewMsg` callback is invoked with the account ID and message ID

### Requirement: Ignore special contacts
The system SHALL ignore messages from special contacts (system messages, device messages). Only messages where `FromId > deltachat.ContactLastSpecial` SHALL be processed.

#### Scenario: System message is ignored
- **WHEN** a message arrives with `FromId <= deltachat.ContactLastSpecial`
- **THEN** the handler takes no action and does not reply

#### Scenario: Regular user message is processed
- **WHEN** a message arrives with `FromId > deltachat.ContactLastSpecial`
- **THEN** the handler proceeds to route the message based on chat type

### Requirement: Group message routing
The system SHALL identify group chat messages and pass them through for processing. In the bootstrap skeleton, group messages are logged but no action is taken (placeholder for future filter engine).

#### Scenario: Group message received
- **WHEN** a message arrives in a group chat from a regular user
- **THEN** the system logs the message receipt and takes no further action

### Requirement: DM help response
The system SHALL reply to direct messages with a static help/usage text explaining what the bot does and how to use it in groups.

#### Scenario: User sends a DM to the bot
- **WHEN** a regular user sends any message directly to the bot (not in a group)
- **THEN** the bot replies with a static help/usage text message

#### Scenario: Help text content
- **WHEN** the bot sends a DM help response
- **THEN** the message SHALL include the bot name, a brief description of its purpose, and instructions to add it to a group

### Requirement: Chat type detection
The system SHALL use the Delta Chat RPC API to determine whether an incoming message belongs to a group chat or a direct message (1:1 chat).

#### Scenario: Group chat identified
- **WHEN** a message arrives and the chat type is `ChatTypeGroup` or `ChatTypeBroadcast`
- **THEN** the system routes it as a group message

#### Scenario: Direct message identified
- **WHEN** a message arrives and the chat type is `ChatTypeSingle`
- **THEN** the system routes it as a DM
