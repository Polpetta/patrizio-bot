.PHONY: project-setup build run test lint docker-build migrate migrate-create sqlc clean doc-activate-venv doc-setup doc-build doc-local

SHELL := /bin/bash

# Binary name
BINARY_NAME=patrizio
DOCKER_IMAGE=patrizio

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
	go test ./...

# Run linter
lint:
	golangci-lint run

# Build Docker image
docker-build:
	docker buildx build -t $(DOCKER_IMAGE) --progress=plain .

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

doc-setup:
	python3 -m venv .venv
	source ./.venv/bin/activate && pip3 install zensical==0.0.27

doc-build:
	source ./.venv/bin/activate && zensical build

doc-local:
	source ./.venv/bin/activate && zensical serve

# Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean
