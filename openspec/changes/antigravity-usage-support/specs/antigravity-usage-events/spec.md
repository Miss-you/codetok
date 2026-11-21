## ADDED Requirements

### Requirement: Antigravity provider emits usage events from conversation databases

The Antigravity provider SHALL read local `conversations/*.db` files and emit one `UsageEvent` per model call recorded in `gen_metadata`.

#### Scenario: Generation row becomes one event

- **WHEN** a conversation database has a `gen_metadata` row whose `ChatModelMetadata.usage` has non-zero counts
- **THEN** the provider returns one `UsageEvent` with `InputOther = input_tokens`, `InputCacheRead = cache_read_tokens`, `InputCacheCreate = cache_write_tokens`, and `Output = output_tokens`
- **AND** the event model is `response_model`, or `model_display_name` when `response_model` is empty

#### Scenario: Steps do not double count

- **WHEN** one generation row lists several step indices and each step carries the same `model_usage`
- **THEN** the provider counts the usage once, from the generation row

#### Scenario: Event time comes from the first step

- **WHEN** a generation row lists step indices
- **THEN** the event timestamp is the first step's `created_at`, or `finished_generating_at` when `created_at` is missing

#### Scenario: Unusable rows are skipped

- **WHEN** a generation row has no usage, all-zero usage, no timestamp, or cannot be decoded
- **THEN** the provider skips that row and keeps the other rows of the file

### Requirement: Antigravity data directories

The provider SHALL scan the CLI root `~/.gemini/antigravity-cli` and the IDE root `~/.gemini/antigravity` by default, and only the given root when a directory override is set.

#### Scenario: Missing default roots

- **WHEN** one or both default roots do not exist
- **THEN** the provider returns events from the roots that exist and no error

#### Scenario: Directory override

- **WHEN** `--antigravity-dir <root>` is set
- **THEN** only `<root>/conversations/*.db` is scanned

### Requirement: Antigravity files are read-only

The provider SHALL NOT write to, lock, or modify Antigravity files, and SHALL NOT call remote APIs.

#### Scenario: Live conversation with WAL data

- **WHEN** a conversation database has a non-empty `-wal` file
- **THEN** the provider reads a temporary copy of the database and WAL, so the latest calls are included without touching the originals

### Requirement: Antigravity sessions

The provider SHALL return one `SessionInfo` per conversation database with counted usage.

#### Scenario: Session totals

- **WHEN** a conversation database has counted generation rows
- **THEN** its session usage is the sum of those rows, start and end times are the earliest and latest event times, turns is the number of counted rows, and the title comes from `conversation_summaries.db` when present
