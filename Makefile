.PHONY: project-setup build run test lint docker-build migrate migrate-create sqlc clean doc-activate-venv doc-setup doc-build doc-local doc-lint

SHELL := /bin/bash

# Binary name
BINARY_NAME=patrizio
DOCKER_IMAGE=patrizio

# Go toolchain version, sourced from go.mod so it stays authoritative.
GO_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)

project-setup: doc-setup
	pre-commit install
	pre-commit install --hook-type pre-push

# Build the binary
build:
	go build -o $(BINARY_NAME) ./cmd/patrizio

# Run the bot (serve mode)
run:
	go run ./cmd/patrizio serve

# Run all tests
test:
	go test ./cmd/... ./internal/...

# Run linter
# Explicit package list to avoid golangci-lint walking into data/ (DeltaChat state).
# Go's ./... only skips dirs prefixed with . or _ (and testdata), so data/ would
# otherwise be traversed and fail with permission-denied on data/chat_state.
lint:
	gofmt -w .
	golangci-lint run ./cmd/... ./internal/...

# Build Docker image
docker-build: Dockerfile
	docker buildx build -t $(DOCKER_IMAGE) --progress=plain --build-arg GO_VERSION=$(GO_VERSION) .

# Update Dockerfile's default ARGs
Dockerfile: go.mod
	@sed -i 's/ARG GO_VERSION.*/ARG GO_VERSION=$(GO_VERSION)/' Dockerfile
	@echo "Updated Dockerfile defaults with GO_VERSION=$(GO_VERSION)"

# Run pending database migrations
migrate:
	go run ./cmd/patrizio migrate

# Create a new migration file
# Usage: make migrate-create NAME=add_filters_table
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	goose -dir migrations create $(NAME) sql

# Regenerate Go code from SQL query files
sqlc:
	sqlc generate

# Phony alias so `make doc-setup` still triggers a check.
# The real work is guarded by a stamp file whose mtime tracks the last
# successful `uv sync`, so the recipe only re-runs when one of the
# prerequisite files actually changes.
doc-setup: .venv/.doc-setup.stamp doc-lint-setup

doc-lint-setup: .git/.doc-lint.stamp

.git/.doc-lint.stamp: vale.ini
	vale sync
	@touch $@

.venv/.doc-setup.stamp: .python-version pyproject.toml uv.lock
	uv sync --locked --all-extras --dev
	@touch $@

doc-build: doc-setup
	uv run zensical build

doc-local: doc-setup
	uv run zensical serve

doc-lint: doc-lint-setup
	vale README.md $$(find docs -type f -name '*.md' -not -path 'docs/superpowers/*')

# Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean
