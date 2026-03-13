# Contributing to Patrizio Bot

## Branching Strategy

We use a simple feature-branch workflow based on Git Flow concepts. The main
branch holds production code. Feature branches merge back into it.

### Example

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
  golangci-lint run ./...
  ```

- **Markdown** – `markdownlint` enforces style. Run with:

  ```bash
  npx markdownlint docs/**/*.md
  ```

- **Formatting** – Run `gofmt -w .` before committing.

## Testing

- Run all unit tests:

  ```bash
  go test ./...
  ```

The repository contains test data in `internal/testdata`. No external services are required.

## Running Locally

```bash
# Build the bot
go build -o patrizio ./cmd/patrizio

# Create required directories
mkdir -p data/media data/db

# Copy default config
cp patrizio.toml config.toml

# Start the bot
./patrizio
```

## Pull-Request Guidelines

1. Keep changes focused to a single feature or bug fix.
2. Include tests for new behaviour.
3. Run CI locally (`act` or `git push`).
4. Provide a clear description and any relevant issue references.
5. Ensure all lint checks pass.

Happy hacking!
