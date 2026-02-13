## MODIFIED Requirements

### Requirement: Group message routing
The system SHALL identify group chat messages and route them through the filter engine for matching. When a match is found, the system SHALL produce the appropriate response (text, media, or reaction) as a quote-reply to the triggering message.

#### Scenario: Group message matches a text filter
- **WHEN** a message arrives in a group chat and matches a text filter trigger
- **THEN** the system quote-replies to the triggering message with the filter's response text

#### Scenario: Group message matches a media filter
- **WHEN** a message arrives in a group chat and matches a media filter trigger
- **THEN** the system quote-replies to the triggering message with the media attachment (image, sticker, GIF, or video)

#### Scenario: Group message matches a reaction filter
- **WHEN** a message arrives in a group chat and matches a reaction filter trigger
- **THEN** the system adds the configured emoji reaction to the triggering message

#### Scenario: Group message matches multiple filters
- **WHEN** a message arrives in a group chat and matches triggers for multiple filters
- **THEN** the system produces responses for all matched filters

#### Scenario: Group message matches no filters
- **WHEN** a message arrives in a group chat and matches no filter triggers
- **THEN** the system takes no action (no reply, no reaction)

#### Scenario: Group message is a filter management command
- **WHEN** a message arrives in a group chat starting with `/filter`, `/stop`, `/stopall`, or `/filters`
- **THEN** the system routes it to the filter management command handler instead of the filter matching engine

### Requirement: Message reception via OnNewMsg
The system SHALL register a `bot.OnNewMsg` callback during `OnBotInit` that receives all incoming messages across all accounts. The handler SHALL have access to the `Dependencies` struct for calling domain logic.

#### Scenario: Incoming message triggers handler
- **WHEN** the bot receives a new message from any chat
- **THEN** the `OnNewMsg` callback is invoked with the account ID and message ID

#### Scenario: Handler has access to dependencies
- **WHEN** the `OnNewMsg` callback is invoked
- **THEN** the handler can access `FilterRepository`, `MediaStorage`, and `Config` via the injected `Dependencies`
