---
icon: lucide/wrench
---

# Usage

Patrizio is a group chat bot that responds to messages based on configured filters. Add Patrizio to a group chat to use it.

## Commands

### /filter <trigger> <response>
Create a text filter. When a message contains the trigger word, Patrizio will reply with the response text.

Examples:
- `/filter hello Hi there!`
- `/filter "good morning" Rise and shine!`

### /filter (<trigger1>, <trigger2>, ...) <response>
Create a filter with multiple triggers for the same response.

Example:
- `/filter (hi, hello, "good morning") Hey!`

### /filter <trigger> react:<emoji>
Create a reaction filter. Patrizio will react to the triggering message with the given emoji.

Example:
- `/filter lol react:😂`

### /filter <trigger>
Create a media filter. Attach an image, sticker, GIF, or video to the command, or reply to a media message. Patrizio will send that media when the trigger matches.

Example:
- `/filter cat` (with an image attached)

### /stop <trigger>
Remove a single trigger.

Examples:
- `/stop hello`
- `/stop "good morning"`

### /stopall
Remove all filters from the current chat.

### /filters
List all active filters in the current chat.

## Notes
- Triggers are matched as whole words anywhere in a message and are case-insensitive.
- Patrizio doesn't respond in direct messages - add it to a group to get started.