## ADDED Requirements

### Requirement: Filter schema migration
The system SHALL include a new goose migration that creates the five filter-related tables: `filters`, `filter_triggers`, `filter_text_resp`, `filter_media_resp`, and `filter_reaction_resp`. Foreign keys SHALL use `ON DELETE CASCADE`. The `filters` table SHALL have an index on `chat_id`. The `filter_triggers` table SHALL have a UNIQUE constraint on `(filter_id, trigger_text)`.

#### Scenario: Migration creates filter tables
- **WHEN** the application starts and the filter migration has not been applied
- **THEN** goose applies the migration and creates all five tables with the correct columns, constraints, and indexes

#### Scenario: Migration is idempotent
- **WHEN** the application starts and the filter migration has already been applied
- **THEN** goose reports the database is up to date and takes no action

### Requirement: Foreign key enforcement
The system SHALL enable SQLite foreign key enforcement by executing `PRAGMA foreign_keys = ON` after opening the database connection. This ensures `ON DELETE CASCADE` and referential integrity constraints are enforced at the database level.

#### Scenario: Foreign keys enforced
- **WHEN** a filter is deleted from the `filters` table
- **THEN** all related rows in `filter_triggers`, `filter_text_resp`, `filter_media_resp`, and `filter_reaction_resp` are automatically deleted via CASCADE

#### Scenario: Invalid foreign key rejected
- **WHEN** an INSERT into `filter_triggers` references a non-existent `filter_id`
- **THEN** the database returns a foreign key constraint error

### Requirement: sqlc query definitions for filter operations
The system SHALL define all filter-related SQL queries in `.sql` files under the `queries/` directory. The `sqlc` tool SHALL generate type-safe Go functions from these files. No inline SQL strings SHALL appear in Go code (except the migration runner and PRAGMA statements).

#### Scenario: Filter queries generated
- **WHEN** a developer runs `make sqlc`
- **THEN** type-safe Go functions are generated for all filter CRUD and matching queries

## MODIFIED Requirements

### Requirement: SQLite connection
The system SHALL connect to a SQLite database at the path specified by the `db_path` configuration value (from TOML config file or `PATRIZIO_DB_PATH` env var), using the pure Go `modernc.org/sqlite` driver via `database/sql`. The connection SHALL enable WAL mode and foreign key enforcement.

#### Scenario: Database file created on first run
- **WHEN** the application starts and the database file does not exist
- **THEN** SQLite creates the database file at the configured path

#### Scenario: Database connection reused
- **WHEN** the application opens a database connection
- **THEN** it returns a `*sql.DB` instance that can be shared across the application

#### Scenario: WAL mode enabled
- **WHEN** the database connection is opened
- **THEN** `PRAGMA journal_mode=WAL` is executed

#### Scenario: Foreign keys enabled
- **WHEN** the database connection is opened
- **THEN** `PRAGMA foreign_keys = ON` is executed
