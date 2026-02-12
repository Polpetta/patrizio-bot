# Deployment

## Purpose

Defines Docker packaging, build tooling, code quality enforcement, and project documentation for Patrizio.

## Requirements

### Requirement: Multi-stage Dockerfile
The project SHALL include a Dockerfile with a multi-stage build: a `golang:1.25` builder stage, a `deltachat-rpc-server` download stage, and a `gcr.io/distroless/static-debian12` runtime stage.

#### Scenario: Docker image builds successfully
- **WHEN** a developer runs `make docker-build`
- **THEN** Docker produces a runnable image containing the static Patrizio binary and the `deltachat-rpc-server` binary

#### Scenario: Runtime image is distroless
- **WHEN** the Docker image is inspected
- **THEN** the base image is `gcr.io/distroless/static-debian12` with no shell, package manager, or unnecessary system utilities

#### Scenario: deltachat-rpc-server is available
- **WHEN** the container starts
- **THEN** the `deltachat-rpc-server` binary is available in PATH and executable

### Requirement: Non-root container execution
The Docker container SHALL run as a non-root user using the distroless `nonroot` user (uid 65534).

#### Scenario: Process runs as non-root
- **WHEN** the container starts
- **THEN** the process runs as user `nonroot` (uid 65534), not root

### Requirement: Makefile with standard targets
The project SHALL include a Makefile with targets for common development and build tasks.

#### Scenario: Build target compiles binary
- **WHEN** a developer runs `make build`
- **THEN** the Go binary is compiled from `./cmd/patrizio`

#### Scenario: Test target runs all tests
- **WHEN** a developer runs `make test`
- **THEN** `go test ./...` runs and reports results

#### Scenario: Lint target runs golangci-lint
- **WHEN** a developer runs `make lint`
- **THEN** `golangci-lint run` executes against the codebase

#### Scenario: Docker build target
- **WHEN** a developer runs `make docker-build`
- **THEN** the multi-stage Docker image is built

#### Scenario: Migration target
- **WHEN** a developer runs `make migrate`
- **THEN** pending database migrations are applied

#### Scenario: Sqlc code generation target
- **WHEN** a developer runs `make sqlc`
- **THEN** Go code is regenerated from `.sql` query files

#### Scenario: Clean target removes artifacts
- **WHEN** a developer runs `make clean`
- **THEN** build artifacts and compiled binaries are removed

### Requirement: Pre-commit hooks
The project SHALL include a `.pre-commit-config.yaml` with hooks for code quality enforcement.

#### Scenario: Lint runs on commit
- **WHEN** a developer commits Go code changes
- **THEN** `golangci-lint run --fix` executes automatically before the commit

#### Scenario: Build check on commit
- **WHEN** a developer commits Go code changes
- **THEN** `go build ./...` executes to verify the code compiles

#### Scenario: Docker build check on commit
- **WHEN** a developer commits Go code changes
- **THEN** `make docker-build` executes to verify the Docker image builds

#### Scenario: Tests run on push
- **WHEN** a developer pushes to a remote
- **THEN** `make test` executes to verify all tests pass

### Requirement: Golangci-lint configuration
The project SHALL include a `.golangci.yml` configuration with a comprehensive set of linters including `govet`, `staticcheck`, `errcheck`, `gosec`, `revive`, and others.

#### Scenario: Strict error checking enabled
- **WHEN** golangci-lint runs
- **THEN** unchecked errors, unchecked type assertions, and errors assigned to blank identifiers are reported

#### Scenario: Test files have relaxed rules
- **WHEN** golangci-lint runs against `_test.go` files
- **THEN** `gocyclo`, `errcheck`, `gosec`, and `funlen` linters are excluded

### Requirement: Gitignore
The project SHALL include a `.gitignore` file that excludes Go build artifacts, SQLite database files, IDE configuration, and other generated files.

#### Scenario: Binary not tracked
- **WHEN** a developer builds the project
- **THEN** the compiled binary is not tracked by git

#### Scenario: Database file not tracked
- **WHEN** the application creates a SQLite database
- **THEN** the `.db` file is not tracked by git

### Requirement: README documentation
The project SHALL include a `README.md` with project overview, prerequisites, setup instructions (including `deltachat-rpc-server` installation), build and run commands, bot initialization flow (`init`, `link`, `serve`), and development guide.

#### Scenario: New developer can set up the project
- **WHEN** a developer reads the README and follows the instructions
- **THEN** they can install dependencies, initialize the bot, and run it locally

### Requirement: License file
The project SHALL include a `LICENSE` file containing the full text of the GNU Affero General Public License version 3 or later (AGPL-3.0-or-later).

#### Scenario: License file present
- **WHEN** a user inspects the repository root
- **THEN** a `LICENSE` file exists containing the AGPL-3.0 license text
