---
icon: lucide/home
---

# Patrizio

Patrizio is a lightweight Delta Chat bot for group chats with reactions and media. Written in Go, it deploys quickly.

Use it to send GIFs when users post "good morning" messages. Supports text, media, and reaction filters for your groups.

## Quick-start

```bash
# Clone the repo
git clone https://github.com/Polpetta/patrizio-bot.git
cd patrizio-bot

# Build the binary
go build -o patrizio ./cmd/patrizio

# Prepare config
cp config.yaml.sample config.yaml

# Run
./patrizio
```

Add Patrizio to a Delta Chat group and use the `/filter` commands.

## Documentation

* [Contributing](CONTRIBUTING.md)
* [Architecture](dev/architecture.md)
* [CI/CD](dev/ci.md)
* [FAQ](faq.md)
* [Changelog](changelog.md)
* [Configuration](dev/config.md)
* [Usage Guide](usage.md)

!!! warning "Patrizio is in early development"

Expect rough edges, bugs, and occasional errors. If you find an unreported issue, please open one in the [issue tracker](https://github.com/Polpetta/patrizio-bot/issues).
