
# Project: Patrizio

## Overview

Patrizio is a **Golang** project for building a [Delta Chat](https://delta.chat) bot, built on top of the [deltabot-cli-go](https://github.com/deltachat-bot/deltabot-cli-go/) library.

The bot's goal is to respond to incoming messages based on specific filters/keywords. Responses can take several forms:
- **Text message** — a plain text reply
- **Reaction** — a message reaction (e.g. emoji)
- **Media** — a sticker, GIF, image, or video attachment

## Tech Stack

- **Language:** Go
- **Bot framework:** `github.com/deltachat-bot/deltabot-cli-go/botcli` — provides CLI scaffolding, bot lifecycle hooks (`OnBotInit`, `OnBotStart`), and logging
- **Delta Chat client:** `github.com/chatmail/rpc-client-go/deltachat` — Go bindings for the Delta Chat RPC API
- **CLI framework:** `github.com/spf13/cobra` (used transitively via deltabot-cli-go)
- **Runtime dependency:** `deltachat-rpc-server` must be available in `PATH` at runtime

## Key Concepts

- The bot is built by creating a `botcli.BotCli` instance, registering event handlers (`OnBotInit`, `OnBotStart`), and calling `cli.Start()`
- Message handling is done via `bot.OnNewMsg()` callbacks registered during `OnBotInit`
- The Delta Chat RPC API is accessed through `bot.Rpc` (e.g. `bot.Rpc.GetMessage()`, `bot.Rpc.MiscSendTextMessage()`)
- The bot is initialized with `<binary> init <email> <password>` and run with `<binary> serve`

## Inspiration: Rose Bot (Telegram)

The filter system is inspired by [Miss Rose](https://missrose.org/docs/filters/) on Telegram. Key behaviors to replicate:

- **Single-word filters** — trigger when a word appears anywhere in a message (e.g. `/filter puppies I love puppies!`)
- **Multi-word filters** — trigger on an exact phrase (e.g. `/filter "I love dogs" I love dogs too!`)
- **Multiple triggers** — define several trigger words/phrases for the same response in one go
- **Media responses** — filters can reply with stickers, images, GIFs, or videos instead of (or in addition to) text
- **Prefix filters** — only match when the trigger is at the start of a message (`prefix:<trigger>`)
- **Exact filters** — only match when the entire message equals the trigger (`exact:<trigger>`)
- **Filter management** — list, add, and remove filters at runtime
