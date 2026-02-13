## ADDED Requirements

### Requirement: Trigger validation
The system SHALL validate that filter trigger text contains only Unicode letters (`\p{L}`), Unicode digits (`\p{N}`), and space characters. Triggers containing any other characters (punctuation, symbols, control characters) SHALL be rejected at creation time. Triggers SHALL be stored lowercased in the database.

#### Scenario: Valid single-word trigger
- **WHEN** a user creates a filter with trigger text `dog`
- **THEN** the trigger is accepted and stored as `dog`

#### Scenario: Valid multi-word trigger
- **WHEN** a user creates a filter with trigger text `I love dogs`
- **THEN** the trigger is accepted and stored as `i love dogs`

#### Scenario: Valid Unicode trigger (Japanese)
- **WHEN** a user creates a filter with trigger text `犬`
- **THEN** the trigger is accepted and stored as `犬`

#### Scenario: Valid Unicode trigger (Cyrillic)
- **WHEN** a user creates a filter with trigger text `Привет`
- **THEN** the trigger is accepted and stored as `привет`

#### Scenario: Trigger with punctuation rejected
- **WHEN** a user creates a filter with trigger text `c++`
- **THEN** the trigger is rejected with an error message explaining only letters, numbers, and spaces are allowed

#### Scenario: Trigger with symbols rejected
- **WHEN** a user creates a filter with trigger text `hello!`
- **THEN** the trigger is rejected with an error message

#### Scenario: Trigger case normalization
- **WHEN** a user creates a filter with trigger text `Hello World`
- **THEN** the trigger is stored as `hello world`

### Requirement: Message normalization
The system SHALL normalize incoming messages before matching by lowercasing the entire text and replacing all characters that are not Unicode letters (`\p{L}`), Unicode digits (`\p{N}`), or whitespace (`\s`) with spaces. This normalization is purely mechanical input sanitization with no business logic.

#### Scenario: Message with trailing punctuation
- **WHEN** a message `I love my dog!` is normalized
- **THEN** the normalized text is `i love my dog `

#### Scenario: Message with embedded punctuation
- **WHEN** a message `dog, cat, and fish` is normalized
- **THEN** the normalized text is `dog  cat  and fish`

#### Scenario: Message with no special characters
- **WHEN** a message `hello world` is normalized
- **THEN** the normalized text is `hello world`

#### Scenario: Message with mixed scripts
- **WHEN** a message `Hello 犬!` is normalized
- **THEN** the normalized text is `hello 犬 `

### Requirement: SQL-based filter matching
The system SHALL match normalized messages against stored triggers entirely in SQL using `INSTR` with space-padding. The query SHALL pad both the normalized message and each trigger with leading and trailing spaces, then check for substring containment. This ensures whole-word matching without false positives from partial word matches.

#### Scenario: Trigger matches word in middle of message
- **WHEN** the trigger `dog` is stored and the normalized message is `i love my dog today`
- **THEN** the trigger matches (` dog ` found inside ` i love my dog today `)

#### Scenario: Trigger matches word at start of message
- **WHEN** the trigger `dog` is stored and the normalized message is `dog is cute`
- **THEN** the trigger matches (` dog ` found inside ` dog is cute `)

#### Scenario: Trigger matches word at end of message
- **WHEN** the trigger `dog` is stored and the normalized message is `i love my dog`
- **THEN** the trigger matches (` dog ` found inside ` i love my dog `)

#### Scenario: Trigger matches entire message
- **WHEN** the trigger `dog` is stored and the normalized message is `dog`
- **THEN** the trigger matches (` dog ` found inside ` dog `)

#### Scenario: Trigger does not match partial word
- **WHEN** the trigger `dog` is stored and the normalized message is `hotdog`
- **THEN** the trigger does not match (` dog ` not found inside ` hotdog `)

#### Scenario: Multi-word trigger matches phrase
- **WHEN** the trigger `i love dogs` is stored and the normalized message is `i love dogs so much`
- **THEN** the trigger matches

#### Scenario: Multi-word trigger does not match words out of order
- **WHEN** the trigger `love dogs` is stored and the normalized message is `dogs love me`
- **THEN** the trigger does not match

### Requirement: Response resolution via CTE and UNION ALL
The system SHALL resolve matched triggers to their responses using a CTE to find matching trigger rows, then UNION ALL across the three response tables (`filter_text_resp`, `filter_media_resp`, `filter_reaction_resp`), joining only the table that matches the filter's `response_type`. Each UNION branch SHALL filter by `response_type` to avoid unnecessary joins.

#### Scenario: Text filter matched
- **WHEN** an incoming message matches a trigger for a filter with `response_type = 'text'`
- **THEN** the query returns the `response_text` from `filter_text_resp`

#### Scenario: Media filter matched
- **WHEN** an incoming message matches a trigger for a filter with `response_type = 'media'`
- **THEN** the query returns the `media_hash` and `media_type` from `filter_media_resp`

#### Scenario: Reaction filter matched
- **WHEN** an incoming message matches a trigger for a filter with `response_type = 'reaction'`
- **THEN** the query returns the `reaction` emoji from `filter_reaction_resp`

#### Scenario: Multiple filters matched
- **WHEN** an incoming message matches triggers for multiple filters in the same chat
- **THEN** the query returns all matching responses

#### Scenario: No filters matched
- **WHEN** an incoming message matches no triggers for the chat
- **THEN** the query returns an empty result set

### Requirement: Filter matching scoped to chat
The system SHALL only match triggers that belong to filters in the same chat as the incoming message. Filters from other chats SHALL NOT be considered.

#### Scenario: Trigger exists in different chat
- **WHEN** chat A has a trigger `hello` and a message containing `hello` arrives in chat B
- **THEN** the trigger does not match
