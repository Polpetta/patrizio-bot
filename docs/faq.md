---
icon: lucide/circle-question-mark
---

# FAQ – Frequently Asked Questions

## Why does my filter not trigger?

* Filters are matched as **whole words**. Use parentheses to group multiple triggers:

  ```bash
  /filter (hi, hello, "good morning") Hey!
  ```

!!! note
    Patrizio uses case-insensitive matching. Finally, make sure you're not using a different character set!

More can be found in its [filter usage guide](user/filters.md)

## What if I want to add a new command?

Please refer to the [Developer documentation](dev/index.md) for further information about Patrizio's architecture.

## How to run tests locally?

Please refer to the [project README](https://github.com/Polpetta/patrizio-bot/blob/main/README.md) for further
information.

## Is there a public instance?

Not yet, due to data being saved in plain without any security. I'll think about a public instance once some sort of
privacy measure has been taken in that sense

## Why does `/prompt` say it's not configured?

The `/prompt` command requires an OpenAI-compatible API key. If the bot operator hasn't set `openai_api_key` in the
configuration, the feature is disabled. Check the [configuration docs](dev/config.md) for setup instructions.

## Why does `/prompt` say I'm not authorized?

The bot operator can restrict which chats are allowed to use the AI feature via the `openai_allowed_chat_ids` setting.
If your chat is not in the list, you'll receive a "not authorized" message. Ask the person running the bot to add your
chat ID to the allowlist.

## Can I use a local AI model instead of OpenAI?

Yes! Patrizio supports any OpenAI-compatible API. You can point it at [Ollama](https://ollama.com/),
[LMStudio](https://lmstudio.ai/), or any other provider by setting `openai_base_url` to the local endpoint. For
example, for Ollama you would set `openai_base_url = "http://localhost:11434/v1"`.

## How do I start a fresh AI conversation?

Each `/prompt` creates an independent thread. To start over, simply send a new `/prompt` message instead of replying to
an existing conversation chain.
