## ADDED Requirements

### Requirement: Daily aggregates usage by event date

The `daily` command SHALL aggregate local token usage using each `provider.UsageEvent.Timestamp` rather than `provider.SessionInfo.StartTime`.

#### Scenario: Same session contributes to multiple days

- **WHEN** a provider returns two usage events from the same session on different localized dates
- **THEN** `daily --json --all` SHALL return one daily row for each event date
- **AND** each row's token usage SHALL include only the events from that date.

#### Scenario: Daily rows use the selected timezone

- **WHEN** a usage event's UTC timestamp falls on different calendar dates in UTC and `Asia/Shanghai`
- **THEN** `daily --json --all --timezone UTC` SHALL use the UTC date
- **AND** `daily --json --all --timezone Asia/Shanghai` SHALL use the Shanghai date.

### Requirement: Daily filters usage events by localized date window

The `daily` command SHALL apply `--since`, `--until`, `--days`, and `--all` to usage events by localized calendar date.

#### Scenario: Explicit date filters use event local dates

- **WHEN** usage events occur around a timezone day boundary
- **AND** the command is run with matching `--since`, `--until`, and `--timezone`
- **THEN** only events whose localized date is inside the inclusive date window SHALL contribute to output rows.

#### Scenario: Default rolling window uses local day boundaries

- **WHEN** `daily --json --timezone Asia/Shanghai` is run without `--since`, `--until`, or `--all`
- **THEN** the default `--days` window SHALL include events from the selected local start date onward
- **AND** it SHALL exclude events before that local start date.

### Requirement: Daily preserves JSON grouping semantics

The `daily` command SHALL preserve existing `DailyStats` JSON semantics while switching from session aggregation to event aggregation.

#### Scenario: CLI grouping fields are stable

- **WHEN** `daily --json --all` aggregates usage events with the default `--group-by cli`
- **THEN** each row's `provider` SHALL remain the provider identity
- **AND** `group_by` SHALL be `cli`
- **AND** `group` SHALL be the provider identity.

#### Scenario: Model grouping across providers is stable

- **WHEN** usage events from multiple providers share one model and `daily --json --all --group-by model` is run
- **THEN** the output SHALL contain one model-grouped row
- **AND** `group_by` SHALL be `model`
- **AND** `group` SHALL be the model name
- **AND** `providers` SHALL list the contributing providers.
