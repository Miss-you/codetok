## ADDED Requirements

### Requirement: Kimi provider emits native usage events

The Kimi provider SHALL implement native usage event collection for local Kimi session logs.

#### Scenario: StatusUpdate token usage becomes one event

- **WHEN** a `wire.jsonl` file contains a valid `StatusUpdate` payload with `token_usage`
- **THEN** Kimi usage event collection returns one `UsageEvent` for that `StatusUpdate`
- **AND** the event token usage equals that payload's token usage values

#### Scenario: StatusUpdate timestamp is preserved

- **WHEN** one Kimi session has `StatusUpdate` records on different timestamps or dates
- **THEN** each returned event uses the timestamp from its own `StatusUpdate` line

#### Scenario: Kimi event metadata matches session metadata behavior

- **WHEN** Kimi metadata or logs provide session ID, title, workdir hash, or model name
- **THEN** each returned Kimi usage event includes the same provider/session metadata that session parsing would use

#### Scenario: Existing Kimi session parsing remains compatible

- **WHEN** Kimi native event collection is added
- **THEN** existing `CollectSessions` and `parseSession` behavior remains unchanged
