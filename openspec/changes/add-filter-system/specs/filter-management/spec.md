## ADDED Requirements

### Requirement: Create text filter with single-word trigger
The system SHALL allow any group member to create a text filter by sending `/filter <word> <reply>` in a group chat. The trigger word SHALL be validated (Unicode alphanumeric and spaces only) and the filter SHALL be stored scoped to that chat.

#### Scenario: Single-word text filter created
- **WHEN** a user sends `/filter puppies I love puppies!` in a group chat
- **THEN** the system creates a filter with trigger `puppies` and response text `I love puppies!` for that chat
- **AND** the system confirms the filter was created

#### Scenario: Invalid trigger rejected
- **WHEN** a user sends `/filter c++ This is C++` in a group chat
- **THEN** the system rejects the filter with a message explaining only letters, numbers, and spaces are allowed

### Requirement: Create text filter with multi-word trigger
The system SHALL allow creating filters with multi-word phrase triggers by quoting the trigger with double quotes: `/filter "<phrase>" <reply>`.

#### Scenario: Multi-word text filter created
- **WHEN** a user sends `/filter "I love dogs" I love dogs too!` in a group chat
- **THEN** the system creates a filter with trigger `i love dogs` and response text `I love dogs too!`

### Requirement: Create filter with multiple triggers at once
The system SHALL allow creating a single filter with multiple triggers by wrapping the trigger list in parentheses with comma separation: `/filter (<one>, <two>, <three>) <reply>`. Quoted phrases SHALL be supported within the parentheses.

#### Scenario: Multiple triggers for one response
- **WHEN** a user sends `/filter (hi, hello, hey) Hello to you too!` in a group chat
- **THEN** the system creates one filter with three triggers (`hi`, `hello`, `hey`) all mapped to response text `Hello to you too!`

#### Scenario: Multiple triggers with quoted phrases
- **WHEN** a user sends `/filter (hi, "hi there", hello) Greetings!` in a group chat
- **THEN** the system creates one filter with three triggers (`hi`, `hi there`, `hello`)

#### Scenario: One invalid trigger in batch rejects entire command
- **WHEN** a user sends `/filter (hi, hello!, hey) Greetings!` in a group chat
- **THEN** the system rejects the entire command because `hello!` contains invalid characters

### Requirement: Create media filter by replying to attachment
The system SHALL allow creating a media filter by replying to a message containing an attachment (image, sticker, GIF, or video) with the `/filter` command. The trigger syntax is the same as text filters, but no reply text is specified.

#### Scenario: Media filter created from image reply
- **WHEN** a user replies to a message containing an image with `/filter dog`
- **THEN** the system creates a filter with trigger `dog` that responds with that image

#### Scenario: Media filter created from sticker reply
- **WHEN** a user replies to a sticker message with `/filter "nice one"`
- **THEN** the system creates a filter with trigger `nice one` that responds with that sticker

### Requirement: Create reaction filter
The system SHALL allow creating a reaction filter by specifying an emoji reaction as the response. The syntax SHALL be `/filter <trigger> react:<emoji>`.

#### Scenario: Reaction filter created
- **WHEN** a user sends `/filter lol react:😂` in a group chat
- **THEN** the system creates a filter with trigger `lol` that reacts with 😂 to triggering messages

### Requirement: Remove a filter trigger
The system SHALL allow removing a filter trigger by sending `/stop <word>` or `/stop "<phrase>"` in a group chat. If the removed trigger was the last trigger on a filter, the entire filter (including its response) SHALL be deleted.

#### Scenario: Single trigger removed
- **WHEN** a filter has triggers `hi` and `hello`, and a user sends `/stop hi`
- **THEN** the trigger `hi` is removed, but the filter and trigger `hello` remain

#### Scenario: Last trigger removed deletes filter
- **WHEN** a filter has only trigger `dog`, and a user sends `/stop dog`
- **THEN** the trigger and the entire filter are deleted

#### Scenario: Non-existent trigger
- **WHEN** a user sends `/stop nonexistent` and no filter in the chat has that trigger
- **THEN** the system responds with a message indicating no filter was found for that trigger

### Requirement: Remove all filters in a chat
The system SHALL allow removing all filters in a group chat by sending `/stopall`.

#### Scenario: All filters removed
- **WHEN** a chat has 5 filters and a user sends `/stopall`
- **THEN** all 5 filters and their triggers and responses are deleted
- **AND** the system confirms the removal

#### Scenario: No filters to remove
- **WHEN** a chat has no filters and a user sends `/stopall`
- **THEN** the system responds indicating there are no filters to remove

### Requirement: List all filters in a chat
The system SHALL allow listing all filters in a group chat by sending `/filters`. The list SHALL show all trigger words/phrases grouped by their associated filter.

#### Scenario: Filters listed
- **WHEN** a chat has 3 filters and a user sends `/filters`
- **THEN** the system responds with a list showing all triggers and their response types

#### Scenario: No filters to list
- **WHEN** a chat has no filters and a user sends `/filters`
- **THEN** the system responds indicating no filters are configured

### Requirement: Any group member can manage filters
The system SHALL NOT restrict filter management commands to admins. Any member of a group chat SHALL be able to create, remove, and list filters.

#### Scenario: Non-admin creates filter
- **WHEN** a non-admin group member sends `/filter hello Hi there!`
- **THEN** the filter is created successfully

#### Scenario: Non-admin removes filter
- **WHEN** a non-admin group member sends `/stop hello`
- **THEN** the filter trigger is removed successfully
