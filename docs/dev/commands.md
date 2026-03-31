---
icon: lucide/terminal
---

# Command Parsing and Validation

All commands are parsed by the pure‑logic layer in `internal/domain/command.go`.
The package exports a small set of command structs – a `FilterCommand`, a
`StopCommand`. It also includes a few constants that represent the
supported commands. For now, all the commands live there since they're quite
simple and it doesn't make sense yet to split into additional separated
packages.

## AI Prompt command

| Command   | What it does                                                          | Example                                  |
|-----------|-----------------------------------------------------------------------|------------------------------------------|
| `/prompt` | Send a message to the AI assistant. Starts a new conversation thread. | `/prompt What is the capital of France?` |

The `/prompt` command is parsed by `ParsePromptCommand`, which simply extracts everything after `/prompt` as the
user's message text. Unlike filter commands, `/prompt` does not go through token extraction or trigger validation — it
passes the raw message content to the AI client.

Thread continuations (replies to Patrizio's AI messages) are detected by `isThreadContinuation` in `handler.go`, not
by the command parser. When a message quotes a known conversation message, it is treated as a continuation without
requiring any command prefix.

## Reaction based commands (aka Filters)

Here is the list of the current implemented filters (note this is not an usage guide, that one can be seen in the [User
guide](../user/index.md)):

| Command    | What it does                                                                                                    | Example                   |
|------------|-----------------------------------------------------------------------------------------------------------------|---------------------------|
| `/filter`  | Create a filter. Trigger can be a word, phrase, or list. Reply can be text, reaction (`react:emoji`), or media. | `/filter hello Hi there!` |
| `/stop`    | Remove a single trigger from the current chat.                                                                  | `/stop hello`             |
| `/stopall` | Remove every filter in the chat.                                                                                | `/stopall`                |
| `/filters` | List all active filters.                                                                                        | `/filters`                |

The parsing logic follows two steps:

1. **Token extraction** – Handles quoted strings and comma‑separated lists. Uses helpers like `parseNextToken` and
   `parseCommaSeparatedTriggers`.
2. **Command construction** – Builds a `FilterCommand` or `StopCommand` struct.

Because the parser is pure, it can be unit‑tested in isolation. `ValidateTrigger` guarantees that trigger text contains
only Unicode letters, digits and spaces. When a trigger is stored it is normalised to lower‑case. Incoming messages are
normalised with `NormalizeMessage` so that matching is case‑insensitive.
