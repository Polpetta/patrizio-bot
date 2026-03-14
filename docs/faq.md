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
