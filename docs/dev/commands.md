---
icon: lucide/terminal
---

# Command Parsing and Validation

All commands are parsed by the pure-logic layer in `internal/domain/command.go`.
The package exports a small set of command structs - a `FilterCommand`, a
`StopCommand`. It also includes a few constants that represent the
supported commands. For now, all the commands live there since they're quite
simple and it doesn't make sense yet to split into additional separated
packages.

## AI Prompt command

| Command   | What it does                                                          | Example                                  |
|-----------|-----------------------------------------------------------------------|------------------------------------------|
| `/prompt` | Send a message to the AI assistant. Starts a new conversation thread. | `/prompt What is the capital of France?` |

The `/prompt` command is parsed by `ParsePromptCommand`, which simply extracts everything after `/prompt` as the
user's message text. Unlike filter commands, `/prompt` does not go through token extraction or trigger validation - it
passes the raw message content to the AI client.

Thread continuations (replies to Patrizio's AI messages) are detected by `isThreadContinuation` in `handler.go`, not
by the command parser. When a message quotes a known conversation message, it is treated as a continuation without
requiring any command prefix.

### AI tool-calling loop

When memory is enabled for a chat, `handlePromptCommand` and `handleThreadContinuation` pass tool descriptors from
`domain.BuildMemoryTools()` to `AIClient.ChatCompletion`. The OpenAI adapter in `internal/adapter/openai/client.go`
runs a multi-turn loop:

1. Send `Messages + Tools` to the API.
2. If the model returns a plain text response → done.
3. If the model returns tool calls → execute each via `domain.MemoryToolHandler.Handle`, append results as
   `ToolMessage`s, loop.
4. Cap at `openai_max_tool_iterations` to prevent infinite loops.

`MemoryToolHandler` dispatches `read_memory`, `append_memory`, and `update_memory` to `MemoryRepository`. It tracks
whether any write-tools were called and sets `ChatResponse.MemoryWritten = true`, which the handler uses to send a
💾 reaction.

The AI call (and any resulting memory writes) is serialized per-chat via `domain.ChatExecutor` so concurrent
`/prompt`s cannot interleave their tool loops.

## Memory commands

| Command          | What it does                              | Example          |
|------------------|-------------------------------------------|------------------|
| `/memory show`   | Display the current memory file contents. | `/memory show`   |
| `/memory clear`  | Delete the memory file.                   | `/memory clear`  |
| `/memory enable` | Enable AI memory for this chat.           | `/memory enable` |
| `/memory disable`| Disable AI memory for this chat.          | `/memory disable`|

`ParseMemoryCommand` in `command.go` strips the `/memory` prefix and maps the remaining word to a `MemorySubCommand`
constant (`MemoryShow`, `MemoryClear`, `MemoryEnable`, `MemoryDisable`). The regex recognizer at the top of
`command.go` includes `/memory` so the dispatcher in `handler.go` routes it before checking for filter matches.

`/memory clear` is also serialized through `ChatExecutor` to avoid interleaving with an in-flight `update_memory`
tool call.

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

1. **Token extraction** - Handles quoted strings and comma-separated lists. Uses helpers like `parseNextToken` and
   `parseCommaSeparatedTriggers`.
2. **Command construction** - Builds a `FilterCommand` or `StopCommand` struct.

Because the parser is pure, it can be unit-tested in isolation. `ValidateTrigger` guarantees that trigger text contains
only Unicode letters, digits and spaces. When a trigger is stored it is normalised to lower-case. Incoming messages are
normalised with `NormalizeMessage` so that matching is case-insensitive.
