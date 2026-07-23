---
icon: lucide/brain
---

# AI Memory

Patrizio can remember facts about your chat across separate `/prompt` conversations. Each chat (group or DM) has its
own private memory file that the AI maintains over time — things like preferences, names, or recurring topics you've
shared.

!!! note
    AI Memory requires the `/prompt` command to be configured. If the bot operator hasn't set up an OpenAI-compatible
    API, neither `/prompt` nor memory will be available.

## What is memory?

Memory is a per-chat markdown file that only the AI can read and write (via tool calls). It persists across separate
`/prompt` threads, so the AI can recall durable facts even in a fresh conversation. Memory works in both group chats
and DMs.

Memory is **enabled by default** for every chat that can use `/prompt`.

## Commands

### `/memory show` — View current memory

Displays the full contents of the chat's memory file. If the AI hasn't written anything yet, you'll see
"(memory is empty)".

### `/memory clear` — Wipe memory

Deletes the memory file for this chat. The AI starts from scratch on the next `/prompt`.

### `/memory enable` — Re-enable memory

Turns memory back on after it was disabled.

### `/memory disable` — Turn off memory

Disables memory for this chat. The AI will no longer have access to memory tools; `/prompt` continues to work
without them. The existing memory file is preserved — use `/memory clear` first if you also want to erase the
content.

## How the AI uses memory

The AI decides when to read or write memory on its own:

- It calls `read_memory` when a question likely benefits from prior context (e.g., "what do I like to drink?").
- It calls `append_memory` or `update_memory` when you share something worth remembering (e.g., "I prefer espresso").

When the AI writes to memory during a turn, Patrizio reacts to your message with a 💾 emoji so you know something
was saved.

You can also nudge the AI explicitly:

- "Remember that I prefer short answers."
- "Forget what I told you about my schedule."
- "What do you know about me?"

## Privacy & limits

!!! warning
    Memory is stored as **plain text** on disk. See the [Data Storage](../dev/storage.md) page for details. Do not
    store sensitive information like passwords or financial data.

- Memory is per-chat — groups and DMs each have their own file.
- The maximum memory size is configured by the bot operator (default: 8 KB). Writes that would exceed the limit
  are rejected; the AI will notify you.
- Memory is never shared across chats.
