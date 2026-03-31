---
icon: lucide/message-square
---

# AI Prompt

The `/prompt` command lets you have conversations with an AI assistant directly inside Delta Chat. It works in both
group chats and DMs. Each `/prompt` starts a fresh conversation thread, and you can continue chatting by simply
replying to Patrizio's messages — no need to type `/prompt` again.

!!! note
    The `/prompt` command is only available if the bot operator has configured an OpenAI-compatible API. If it hasn't
    been set up, Patrizio will let you know.

## Commands

### `/prompt <message>` - Starting a conversation

Send a message to the AI assistant. Patrizio will reply with the AI's response, quoting your original message.

Examples:

- `/prompt What is the capital of France?`
- `/prompt Explain quantum computing in simple terms`

### Replying to continue - Continuing a conversation

To keep the conversation going, reply to any of Patrizio's AI responses. The full conversation history is sent as
context, so the AI remembers what you've been talking about.

Example:

- You: `/prompt Tell me about penguins`
- Patrizio: _"Penguins are flightless birds..."_
- You: (reply to Patrizio's message) `What do they eat?`
- Patrizio: _"Penguins primarily eat fish..."_

You don't need to type `/prompt` again — just reply to any message in the thread.

### Starting a new conversation

Each `/prompt` creates an independent thread. If you want to change topic or start fresh, simply send a new `/prompt`
instead of replying to an existing chain.

## Notes

- Multiple independent conversations can happen in the same chat at the same time.
- The bot operator can restrict which chats are allowed to use `/prompt` via the `openai_allowed_chat_ids`
  configuration. If your chat is not in the allowlist, you'll receive a "not authorized" message.
- Conversation history has a configurable maximum depth. Very long threads will only send the most recent messages as
  context to the AI.
- The AI provider and model depend on the bot operator's configuration. Patrizio supports any OpenAI-compatible API,
  including OpenAI, Ollama, LMStudio and others.
