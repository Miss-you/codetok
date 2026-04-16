## ADDED Requirements

### Requirement: Commands collect usage events through a native-first bridge

The command layer SHALL provide helpers that collect `provider.UsageEvent` values from registered providers while preserving existing provider filtering and directory override semantics.

#### Scenario: Native event provider is used unchanged

- **GIVEN** a provider implements `provider.UsageEventProvider`
- **WHEN** command code collects usage events
- **THEN** it SHALL call `CollectUsageEvents` for that provider
- **AND** it SHALL append returned events without rewriting their fields
- **AND** it SHALL NOT call `CollectSessions` for that provider.

#### Scenario: Legacy session provider falls back to synthetic events

- **GIVEN** a provider implements only `provider.Provider`
- **WHEN** command code collects usage events
- **THEN** it SHALL call `CollectSessions`
- **AND** it SHALL synthesize one `provider.UsageEvent` per `provider.SessionInfo`
- **AND** the synthetic event SHALL copy provider, model, session ID, title, workdir hash, start time, and token usage from the session.

#### Scenario: Provider selection and directory overrides are preserved

- **GIVEN** command flags include `--provider`, `--base-dir`, or a provider-specific `--<name>-dir`
- **WHEN** command code collects usage events
- **THEN** it SHALL use the same filtering and directory precedence as session collection.

#### Scenario: Missing local directories are skipped

- **GIVEN** a provider returns an `os.IsNotExist` error
- **WHEN** command code collects usage events
- **THEN** it SHALL skip that provider and continue collecting other providers.

### Requirement: Daily resolves date windows in the selected timezone

The `daily` command SHALL expose timezone resolution helpers for date filtering.

#### Scenario: Empty timezone uses the local timezone

- **GIVEN** `--timezone` is empty
- **WHEN** the command resolves the timezone
- **THEN** it SHALL use `time.Local`.

#### Scenario: IANA timezone names are accepted

- **GIVEN** `--timezone Asia/Shanghai`
- **WHEN** the command resolves the timezone
- **THEN** it SHALL load the named IANA location successfully.

#### Scenario: Invalid timezone names fail clearly

- **GIVEN** `--timezone not/a-zone`
- **WHEN** the command resolves the timezone
- **THEN** it SHALL return an error containing `invalid --timezone`.

#### Scenario: Default daily windows use local day boundaries

- **GIVEN** a selected timezone
- **WHEN** `daily` resolves the default `--days` window
- **THEN** the start time SHALL be midnight in the selected timezone for the start date.

#### Scenario: Explicit daily date filters use the selected timezone

- **GIVEN** `--since` or `--until` date filters and a selected timezone
- **WHEN** `daily` resolves the date range
- **THEN** it SHALL parse those dates in the selected timezone
- **AND** `--until` SHALL include the full selected calendar day.
