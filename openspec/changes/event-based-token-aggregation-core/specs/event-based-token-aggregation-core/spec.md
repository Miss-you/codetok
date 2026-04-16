## ADDED Requirements

### Requirement: Shared usage event model

The provider package SHALL expose a timestamped usage event model for local token usage records and SHALL expose an optional provider interface for native event collectors.

#### Scenario: Provider emits usage events

- **WHEN** a provider implements native event collection
- **THEN** it can return `UsageEvent` records containing provider name, model name, session metadata, timestamp, token usage, source path, and event ID

### Requirement: Event daily aggregation

The stats package SHALL aggregate usage events into `DailyStats` rows using each event timestamp in the requested timezone.

#### Scenario: One session spans two local dates

- **WHEN** two events from the same session occur on different localized calendar dates
- **THEN** event aggregation returns one daily row per event date with the corresponding token totals

#### Scenario: Timezone changes date key

- **WHEN** an event timestamp maps to different calendar dates in UTC and another timezone
- **THEN** event aggregation uses the date key from the requested timezone

### Requirement: Event session counting

The stats package SHALL count distinct contributing sessions per date/group rather than counting token events.

#### Scenario: Multiple events from one session on one date

- **WHEN** multiple events share the same session identifier on the same date and group
- **THEN** the aggregated `DailyStats.Sessions` value counts that session once

#### Scenario: Event has no session identifier

- **WHEN** an event has no session identifier but has a source path
- **THEN** the aggregation uses source path as the contributing session key

### Requirement: Event date filtering

The stats package SHALL filter usage events by inclusive localized date keys.

#### Scenario: Filter by since and until dates

- **WHEN** events occur before, inside, and after the requested date range in the selected timezone
- **THEN** only events whose localized date key falls within the inclusive range are returned
