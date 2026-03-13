# CI/CD Pipeline

Patrizio uses GitHub Actions for continuous integration and deployment.

## Workflow Overview

| Step | Description |
|------|-------------|
| `Lint` | Run `markdownlint` and `golangci-lint`.
| `Test` | Execute Go tests.
| `Build` | Compile the binary for Linux and macOS.
| `Release` | Create a GitHub release if a tag is pushed.

## YAML Configuration

The workflow is defined in `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
    tags: ["v*.*.*"]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Install golangci-lint
        run: |
      curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |
      sh -s -- -b $(go env GOPATH)/bin v1.58.0
      - name: Run golangci-lint
        run: golangci-lint run ./...
      - name: Install markdownlint
        run: npm install -g markdownlint-cli
      - name: Run markdownlint
        run: markdownlint docs/**/*.md

  test:
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Run tests
        run: go test ./...

  build:
    runs-on: ubuntu-latest
    needs: test
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Build binary
        run: go build -o patrizio ./cmd/patrizio
      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: patrizio-${{ matrix.os }}
          path: patrizio

  release:
    runs-on: ubuntu-latest
    needs: build
    if: startsWith(github.ref, 'refs/tags/')
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Download artifacts
        uses: actions/download-artifact@v4
        with:
          name: patrizio-ubuntu-latest
      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          files: patrizio
          prerelease: false
```

## Secrets

* `GH_TOKEN` – used by the release job to publish to GitHub.

> The workflow assumes a **release** branch is protected and only merged after all checks pass.

For more details, see the `ci.yml` file in the repository.
