# Bot Lifecycle

The deltabot‑cli‑go framework powers the bot.  `internal/bot/bot.go` registers two callbacks:

* **OnBotInit** – called once when the bot starts.  It’s mainly a hook for future extensions.
* **OnNewMsg** – invoked for every message received.  This is where the bot does its work.

## Startup

During `OnBotInit` the bot registers the `OnNewMsg` handler.  No other side effects happen at this stage.

## Handling a Message

When a message arrives the handler does the following:

* Skips system messages (special contacts, device messages) – see the check in `handler.go`.
* Retrieves chat metadata via the Delta Chat RPC client.
* Decides whether the chat is a group or a direct message.

### Group chats

For groups the handler first looks for bot commands (`/filter`, `/stop`,
etc.). If the text is a command, it’s parsed by the domain code. If it’s not
a command, the message is normalised (lower‑cased, punctuation removed) and
the repository is queried for matching filters. Every matching filter triggers
a reply: text, media, or a reaction, with media files fetched from the storage
adapter.

### Direct chats

In a one‑to‑one conversation the bot simply replies with a short help message – the logic is intentionally minimal.

---

## File references

* `internal/bot/bot.go`
* `internal/bot/handler.go`
