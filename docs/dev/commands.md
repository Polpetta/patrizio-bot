# Command Parsing and Validation

All commands are parsed by the pure‑logic layer in `internal/domain/command.go`.
The package exports a small set of command structs – a `FilterCommand`, a `StopCommand`.
It also includes a few constants that represent the four supported commands.

| Command   | What it does | Example |
|----------|--------------|---------|
| `/filter` | Create a filter. Trigger can be a word, phrase, or list.
Reply can be text, reaction (`react:emoji`), or media. | `/filter hello Hi there!` |
| `/stop`   | Remove a single trigger from the current chat. |
`/stop hello` |
| `/stopall`| Remove every filter in the chat. | `/stopall` |
| `/filters`| List all active filters. | `/filters` |

The parsing logic follows two steps:

1. **Token extraction** – Handles quoted strings and comma‑separated lists.
   Uses helpers like `parseNextToken` and `parseCommaSeparatedTriggers`.
2. **Command construction** – Builds a `FilterCommand` or `StopCommand` struct.

Because the parser is pure, it can be unit‑tested in isolation.
`ValidateTrigger` guarantees that trigger text contains only Unicode letters, digits and spaces.
When a trigger is stored it is normalised to lower‑case.
Incoming messages are normalised with `NormalizeMessage` so that matching is case‑insensitive.

---

## Key functions

* `ParseFilterCommand` – Handles all `/filter` syntaxes. (lines 38‑112)
* `ParseStopCommand` – Handles `/stop` syntax. (lines 114‑138)
* `ValidateTrigger` – Ensures triggers only contain Unicode letters, digits, or spaces. (lines 28‑34 in `filter.go`)

---

## File references

* `internal/domain/filter.go`
* `internal/domain/command.go`
