## ADDED Requirements

### Requirement: Codex emits native usage events
The Codex provider SHALL implement native usage event collection for local rollout JSONL files while preserving existing session collection.

#### Scenario: Token count emits usage event
- **WHEN** a Codex rollout file contains a valid `event_msg` with `type` set to `token_count`
- **THEN** native event collection returns a `UsageEvent` with provider name `codex`, the token event timestamp, token usage, source path, and available session metadata

### Requirement: Codex token delta recovery
The Codex provider SHALL convert Codex token usage payloads into non-zero incremental token deltas.

#### Scenario: Last token usage is present
- **WHEN** a Codex `token_count` record contains `last_token_usage`
- **THEN** the provider emits usage from `last_token_usage` without subtracting the previous cumulative total

#### Scenario: Only cumulative total usage is present
- **WHEN** a Codex `token_count` record contains `total_token_usage` and no `last_token_usage`
- **THEN** the provider emits the difference from the previous cumulative total in the same rollout file

#### Scenario: Cumulative counter resets
- **WHEN** a later Codex `total_token_usage` counter is lower than the previous cumulative counter
- **THEN** the provider treats the lower total as a reset and emits the current total as a fresh delta

### Requirement: Codex event metadata stability
The Codex provider SHALL attach stable session and model metadata to native usage events.

#### Scenario: Multiple session metadata records
- **WHEN** a Codex rollout file contains multiple `session_meta` records
- **THEN** native usage events use the first non-empty session identifier from the file

#### Scenario: Token record lacks model metadata
- **WHEN** a Codex `token_count` record has no valid model metadata and a previous event contains a valid model context
- **THEN** the emitted usage event uses the active model context

### Requirement: Codex default source resolution
The Codex provider SHALL resolve its default local session root from explicit input, `CODEX_HOME`, then the user home fallback.

#### Scenario: Empty base directory with CODEX_HOME
- **WHEN** Codex collection is called with an empty base directory and `CODEX_HOME` is set
- **THEN** the provider scans `$CODEX_HOME/sessions`

#### Scenario: Explicit base directory
- **WHEN** Codex collection is called with an explicit base directory and `CODEX_HOME` is set
- **THEN** the provider scans the explicit base directory
