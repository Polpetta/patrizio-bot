# Message Routing Testability

## Purpose

Defines requirements for the `processMessage` function — the synchronous, testable entry-point for the per-message routing pipeline that `newMsgHandler` launches in a goroutine.

## Requirements

### Requirement: GetMessage failure is logged and dropped
The system SHALL log an error and take no further action when the RPC call to fetch the message fails.

#### Scenario: GetMessage returns an error
- **GIVEN** `processMessage` is invoked for a message ID
- **WHEN** `rpc.GetMessage` returns an error
- **THEN** the error is logged via `logger.Errorf`
- **AND** no message, reaction, or further RPC call is made

### Requirement: Special-contact messages are silently ignored
The system SHALL drop messages from special contacts without logging or replying.

#### Scenario: Message from special contact
- **GIVEN** `rpc.GetMessage` succeeds and `msg.FromId <= deltachat.ContactLastSpecial`
- **WHEN** `processMessage` handles the message
- **THEN** no error is logged and no reply is sent

### Requirement: GetBasicChatInfo failure is logged and dropped
The system SHALL log an error and take no further action when the RPC call to fetch chat info fails.

#### Scenario: GetBasicChatInfo returns an error
- **GIVEN** the message is from a regular user
- **WHEN** `rpc.GetBasicChatInfo` returns an error
- **THEN** the error is logged via `logger.Errorf`
- **AND** no message or reaction is sent

### Requirement: Group messages are dispatched to group handler
The system SHALL route messages from group-type chats to `handleGroupMessage`.

#### Scenario: Message in a group chat
- **GIVEN** `chatInfo.ChatType` is `ChatGroup` (or any group variant)
- **WHEN** `processMessage` handles the message
- **THEN** the group handler is invoked (observable via filter repository or RPC calls)

### Requirement: DM messages are dispatched to DM handler
The system SHALL route messages from single-type chats to `handleDMMessage`.

#### Scenario: Message in a direct message chat
- **GIVEN** `chatInfo.ChatType` is `ChatSingle`
- **WHEN** `processMessage` handles the message
- **THEN** the DM handler is invoked (observable via help text sent or AI client called)

### Requirement: Unknown chat type is warned and dropped
The system SHALL log a warning and take no further action for unrecognised chat types.

#### Scenario: Unknown chat type
- **GIVEN** `chatInfo.ChatType` is a value not handled by the switch statement
- **WHEN** `processMessage` handles the message
- **THEN** a warning is logged via `logger.Warnf`
- **AND** no message or reaction is sent
