## ADDED Requirements

### Requirement: Daily aggregates usage events by localized event date
The `daily` command SHALL aggregate token usage using `provider.UsageEvent.Timestamp` in the selected timezone rather than `provider.SessionInfo.StartTime`.

#### Scenario: One session has usage events on two dates
- **WHEN** a provider emits two usage events for the same session on different localized calendar dates
- **THEN** `daily --json` SHALL return separate daily rows for those dates
- **AND** each row SHALL contain only the token usage from events on that row's date.

### Requirement: Daily filters usage events by selected timezone date keys
The `daily` command SHALL apply `--since`, `--until`, and default `--days` windows to localized event date keys in the selected timezone.

#### Scenario: Timezone shifts an event into the requested date
- **WHEN** an event timestamp is `2026-04-15T18:00:00Z`
- **AND** the user runs `daily --since 2026-04-16 --until 2026-04-16 --timezone Asia/Shanghai`
- **THEN** the event SHALL be included in a row dated `2026-04-16`.

#### Scenario: Default lookback uses local midnight
- **WHEN** the selected timezone maps UTC midnight differently from local midnight
- **THEN** the default `--days` window SHALL include events from the selected local day and exclude events before that local day.

### Requirement: Daily preserves output and flag contracts
The `daily` command SHALL preserve existing JSON field semantics, grouping dimensions, flag conflicts, dashboard-only unit scaling, and local-only provider collection behavior while switching to event aggregation.

#### Scenario: Model grouping spans providers
- **WHEN** events from multiple providers share one model and the user runs `daily --group-by model --json`
- **THEN** the row SHALL keep `group_by` as `model`
- **AND** `group` SHALL contain the model name
- **AND** `provider` SHALL remain empty when the model group spans multiple providers
- **AND** `providers` SHALL list the contributing providers.

#### Scenario: Date-window flag conflicts are reported before collection
- **WHEN** a user provides a conflicting daily date-window flag combination
- **THEN** the command SHALL return the flag validation error
- **AND** provider collection errors SHALL NOT mask that validation error.
