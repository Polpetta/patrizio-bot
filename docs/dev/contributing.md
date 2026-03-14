---
icon: lucide/git-pull-request-create
---

# Contributing to Patrizio Bot

## Branching Strategy

We use a simple feature-branch workflow based on Git Flow concepts. The `main` branch holds production code. Feature
branches merge back into it. If you want to try your work, for each commit a Docker image will be generated

### Example

Thereby you can find an example of development workflow, altough you maybe already accustomed to it:

```bash
# Start from main
git checkout main
git pull

# Create a new feature branch
git checkout -b feature/expand-filters

# Commit your changes
git add .
git commit -m "Add expand filters"

# Push the branch
git push --set-upstream origin feature/expand-filters
```

When the feature is complete, open a PR against **main** and let the CI run.

## Code Style & Linting

- **Go** – `golangci-lint` is run by CI. Run locally with:

  ```bash
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  make lint
  ```

- **Markdown** – `markdownlint` enforces style. Run with:

  ```bash
  pre-commit run markdownlint --run-all-files
  ```

## Testing

- Run all the tests:

  ```bash
  make test
  ```

## Running Locally

```bash
# Build the bot
make build

# Create required directories
mkdir -p ./data/{media,db}

# Configure the bot by editing patrizio.toml (this is the file the app reads)

# Start the bot
./patrizio
```

## Pull-Request Guidelines

When opening up contributions, please:

1. Keep changes focused to a single feature or bug fix.
2. Include tests for new behaviour.
3. Run pre-commit checks locally. They will take care of covering most if not all the checks that are then run in the CI
4. Provide a clear description and any relevant issue references.
5. Ensure all lint checks pass.

!!! note "About AI"
    If you have vibe-coded or you have committed through the use of AI agentic coding (aka Claude Code,
    OpenCode and similar) please disclose it in the PR. This is not a problem per se, but it is fair to let know other
    people about that, so that they can decide whenever they want to use AI assisted software or not.

Happy hacking!
