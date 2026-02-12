# Database

## Purpose

Defines the database layer including SQLite connectivity, schema migration strategy, and SQL query management for Patrizio.

## Requirements

### Requirement: SQLite connection
The system SHALL connect to a SQLite database at the path specified by the `PATRIZIO_DB_PATH` configuration value, using the pure Go `modernc.org/sqlite` driver via `database/sql`.

#### Scenario: Database file created on first run
- **WHEN** the application starts and the database file does not exist
- **THEN** SQLite creates the database file at the configured path

#### Scenario: Database connection reused
- **WHEN** the application opens a database connection
- **THEN** it returns a `*sql.DB` instance that can be shared across the application

### Requirement: Forward-only migrations with goose
The system SHALL use `goose` to manage database schema migrations. Only forward (up) migrations SHALL be supported. Down migrations SHALL NOT be used.

#### Scenario: Pending migrations run at startup
- **WHEN** the application starts and there are pending migration files
- **THEN** goose applies all pending up migrations in version order

#### Scenario: No pending migrations
- **WHEN** the application starts and all migrations have been applied
- **THEN** goose reports that the database is up to date and takes no action

#### Scenario: Migration files embedded in binary
- **WHEN** the application is built
- **THEN** migration SQL files from the `migrations/` directory are embedded via `go:embed` and available at runtime without filesystem access

### Requirement: Migration file format
Migration files SHALL be SQL files in the `migrations/` directory, named with a numeric version prefix (e.g. `001_initial.up.sql`). Each migration file SHALL contain only forward (up) migration statements.

#### Scenario: New migration created
- **WHEN** a developer runs `make migrate-create`
- **THEN** a new migration file is created in `migrations/` with the next version number

### Requirement: SQL query files separated from Go code
All SQL queries SHALL be defined in `.sql` files under the `queries/` directory (one file per query or per domain). Go code SHALL NOT contain inline SQL strings. The `sqlc` tool SHALL generate type-safe Go code from these query files.

#### Scenario: Query file generates Go code
- **WHEN** a developer creates or modifies a `.sql` file in `queries/` and runs `make sqlc`
- **THEN** `sqlc` generates corresponding Go functions with proper types in `internal/database/queries/`

#### Scenario: No inline SQL in Go source
- **WHEN** a developer inspects the Go source code
- **THEN** no SQL query strings are found in `.go` files (except for the embedded migration runner)

### Requirement: Initial migration
The bootstrap SHALL include an initial migration (`001_initial.up.sql`) that creates the base schema. For the skeleton, this creates a minimal placeholder table to verify the migration system works.

#### Scenario: Initial migration applied
- **WHEN** the application starts against a fresh database
- **THEN** the initial migration runs and creates the expected schema
