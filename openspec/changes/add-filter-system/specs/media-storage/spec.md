## ADDED Requirements

### Requirement: Content-addressed storage with SHA-512
The system SHALL store media files on disk using the SHA-512 hex digest of the file content as the filename. The storage directory SHALL be configurable via the `media_path` configuration setting.

#### Scenario: Media file saved with hash filename
- **WHEN** a media file with content hashing to `a1b2c3...` (SHA-512 hex) is saved
- **THEN** the file is written to `<media_path>/a1b2c3...`

#### Scenario: File path deterministic from content
- **WHEN** two identical files are provided
- **THEN** both produce the same SHA-512 hash and therefore the same file path

### Requirement: Deduplication on write
The system SHALL deduplicate media files by content hash. Writing a file with the same SHA-512 hash as an existing file SHALL overwrite the existing file with identical content (idempotent write). No error SHALL be raised.

#### Scenario: First write of unique media
- **WHEN** a media file with a new hash is saved
- **THEN** the file is written to disk

#### Scenario: Duplicate media write is idempotent
- **WHEN** a media file with hash `abc123` already exists on disk and a new file with the same content is saved
- **THEN** the file is overwritten with identical content and no error is raised

### Requirement: Transactional media cleanup on filter deletion
The system SHALL delete media files from disk inside the SQLite write transaction when a filter is deleted, but only if no other filter references the same `media_hash`. The check and deletion SHALL happen before the transaction is committed to prevent race conditions.

#### Scenario: Media file deleted when last reference removed
- **WHEN** a filter with `media_hash = abc123` is deleted and no other `filter_media_resp` row references `abc123`
- **THEN** the file `<media_path>/abc123` is deleted from disk inside the transaction

#### Scenario: Media file preserved when other references exist
- **WHEN** a filter with `media_hash = abc123` is deleted but another filter also references `abc123`
- **THEN** the file `<media_path>/abc123` is NOT deleted from disk

#### Scenario: Concurrent delete and insert race prevented
- **WHEN** thread A deletes the last filter referencing `media_hash = abc123` and thread B simultaneously creates a new filter referencing the same hash
- **THEN** the SQLite write lock serializes the operations: thread A completes its transaction (deleting the file) before thread B starts, and thread B writes the file fresh after its insert commits

### Requirement: Afero VFS abstraction
The system SHALL use the `afero.Fs` interface from `github.com/spf13/afero` for all filesystem operations related to media storage. Production SHALL use `afero.OsFs` and tests SHALL use `afero.MemMapFs`.

#### Scenario: Production uses real filesystem
- **WHEN** the application starts in production
- **THEN** media storage uses `afero.NewOsFs()` to read and write files on the real filesystem

#### Scenario: Tests use in-memory filesystem
- **WHEN** tests exercise media storage logic
- **THEN** media storage uses `afero.NewMemMapFs()` with no files written to disk

### Requirement: MediaStorage port interface
The system SHALL define a `MediaStorage` port interface in the domain package with operations: `Save(hash string, data []byte) error`, `Delete(hash string) error`, `Read(hash string) ([]byte, error)`, `Exists(hash string) (bool, error)`. The Afero-backed adapter SHALL implement this interface.

#### Scenario: Save writes to configured path
- **WHEN** `Save("abc123", data)` is called
- **THEN** the data is written to `<media_path>/abc123` via the `afero.Fs` instance

#### Scenario: Delete removes file
- **WHEN** `Delete("abc123")` is called
- **THEN** the file at `<media_path>/abc123` is removed via the `afero.Fs` instance

#### Scenario: Read returns file content
- **WHEN** `Read("abc123")` is called and the file exists
- **THEN** the content of `<media_path>/abc123` is returned

#### Scenario: Exists checks file presence
- **WHEN** `Exists("abc123")` is called
- **THEN** it returns `true` if the file exists, `false` otherwise
