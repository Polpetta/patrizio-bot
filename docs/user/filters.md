---
icon: lucide/list-filter
---

# Usage

Filters are a powerful functionality to add a custom reply, send media or react to messages that contain certain text.
This is the first feature of the bot, and it took inspiration directly from [Miss Rose](https://missrose.org/) Telegram
bot.

## Commands

### `/filter trigger response` - Creating a text filter

Create a text filter. When a message contains the trigger word, Patrizio will reply with the response text.

Examples:

- `/filter hello Hi there!`
- `/filter "good morning" Rise and shine!`

### `/filter (trigger1, trigger2, ...) response` - Creating multiple text filters

Create a filter with multiple triggers for the same response.

Example:

- `/filter (hi, hello, "good morning") Hey!`

### `/filter trigger react:emoji` - Creating a message reaction

Create a reaction filter. Patrizio will react to the triggering message with the given emoji.

Example:

- `/filter lol react:😂`

### `/filter trigger` - Creating a media filter

Create a media filter. Attach an image, sticker, GIF, or video to the command,
or reply to a media message. Patrizio will send that media when the trigger matches.

Example:

- `/filter cat` (with an image attached)

### `/stop trigger` - Removing a filter

Remove a single trigger.

Examples:

- `/stop hello`
- `/stop "good morning"`

### `/stopall` - Removing all the filters

Remove all filters from the current chat.

### `/filters` - Listing all the filters

List all active filters in the current chat.

## Notes

- Triggers are matched as whole words anywhere in a message and are case-insensitive.
- Filters are available only in group chats.
