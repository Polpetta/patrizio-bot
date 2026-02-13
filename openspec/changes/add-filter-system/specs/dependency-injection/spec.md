## ADDED Requirements

### Requirement: Dependencies struct for dependency injection
The system SHALL define a `Dependencies` struct that holds all port implementations needed by the domain layer. This struct SHALL be constructed at the application entrypoint and passed through to all components that need access to external resources.

#### Scenario: Production dependencies wired at startup
- **WHEN** the application starts
- **THEN** `main.go` constructs a `Dependencies` struct with real adapter implementations (SQLite `FilterRepository`, Afero `MediaStorage`, Viper+TOML `Config`)

#### Scenario: Test dependencies use fakes
- **WHEN** tests exercise domain logic
- **THEN** tests construct a `Dependencies` struct with in-memory SQLite, `afero.MemMapFs`-backed `MediaStorage`, and a directly-constructed `Config` struct

### Requirement: Domain port interfaces
The system SHALL define the following port interfaces in the domain package: `FilterRepository` (filter CRUD and matching queries), `MediaStorage` (file save/delete/read/exists), and `Config` (read configuration values). The domain package SHALL NOT import any adapter packages (no Delta Chat, Afero, or SQLite imports).

#### Scenario: FilterRepository interface defined
- **WHEN** a developer inspects the domain package
- **THEN** the `FilterRepository` interface is defined with methods: `CreateTextFilter`, `CreateMediaFilter`, `CreateReactionFilter`, `RemoveTrigger`, `RemoveAllFilters`, `ListFilters`, `FindMatchingFilters`

#### Scenario: MediaStorage interface defined
- **WHEN** a developer inspects the domain package
- **THEN** the `MediaStorage` interface is defined with methods: `Save`, `Delete`, `Read`, `Exists`

#### Scenario: Config interface defined
- **WHEN** a developer inspects the domain package
- **THEN** the `Config` interface is defined with methods to access `DBPath`, `LogLevel`, and `MediaPath`

#### Scenario: Domain has no external adapter imports
- **WHEN** a developer inspects the import statements of the domain package
- **THEN** no imports from `deltachat`, `afero`, `modernc.org/sqlite`, or `viper` are present

### Requirement: Hexagonal architecture with adapter separation
The system SHALL organize code following hexagonal architecture: the domain package defines ports (interfaces) and core logic; adapter packages implement ports using concrete technologies; the bot package acts as the inbound adapter translating Delta Chat events into domain calls and domain responses into RPC calls.

#### Scenario: Bot as inbound adapter
- **WHEN** a group message arrives via Delta Chat
- **THEN** the bot adapter normalizes the message, calls domain logic via port interfaces, and translates the returned response struct into the appropriate Delta Chat RPC call

#### Scenario: SQLite as outbound adapter
- **WHEN** the domain calls `FilterRepository.FindMatchingFilters`
- **THEN** the SQLite adapter executes the CTE + UNION ALL query and returns domain model structs

#### Scenario: Afero as outbound adapter
- **WHEN** the domain calls `MediaStorage.Save`
- **THEN** the Afero adapter writes the file to the configured path using the injected `afero.Fs` instance

### Requirement: Mockery available for test mock generation
The system SHALL support using Mockery (`github.com/vektra/mockery`) to generate mock implementations of port interfaces when needed for testing. Usage of Mockery is optional — in-memory real implementations (e.g., in-memory SQLite, `MemMapFs`) are preferred where practical. Mockery SHALL be used when a real in-memory implementation is impractical or when verifying specific call sequences is important.

#### Scenario: Mock generated for port interface
- **WHEN** a developer runs Mockery against a port interface
- **THEN** a mock implementation is generated that allows test assertions on method calls, arguments, and return values

#### Scenario: Real fakes preferred over mocks
- **WHEN** a test needs a `FilterRepository`
- **THEN** an in-memory SQLite instance with the real schema is preferred over a Mockery-generated mock, unless call verification is specifically needed
