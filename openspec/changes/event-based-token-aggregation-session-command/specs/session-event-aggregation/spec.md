## ADDED Requirements

### Requirement: Session filters usage by event date

The `session` command SHALL filter local token usage using each
`provider.UsageEvent.Timestamp` rather than `provider.SessionInfo.StartTime`.

#### Scenario: Session started before range has in-range usage

- **WHEN** a provider returns usage events from the same session on different dates
- **AND** `session --json --since <date> --until <date>` selects only the later date
- **THEN** the session SHALL be included
- **AND** its token usage SHALL include only events from the selected date.

### Requirement: Session uses selected timezone for event dates

The `session` command SHALL accept `--timezone` as an IANA timezone name and use it
for date filtering and displayed session dates.

#### Scenario: Event crosses UTC and local date boundaries

- **WHEN** an event timestamp falls on different dates in UTC and `Asia/Shanghai`
- **THEN** `session --timezone UTC` SHALL filter and display by the UTC date
- **AND** `session --timezone Asia/Shanghai` SHALL filter and display by the Shanghai date.

### Requirement: Session groups filtered events by provider session

The `session` command SHALL group filtered usage events by provider and stable session
identity while preserving existing output shape.

#### Scenario: Same session has multiple included events

- **WHEN** multiple included events share one provider and session ID
- **THEN** the command SHALL emit one session row
- **AND** `date` SHALL be the first included event date
- **AND** internal end time SHALL track the latest included event.

#### Scenario: Same session ID appears in multiple providers

- **WHEN** two providers return the same session ID
- **THEN** the command SHALL emit separate session rows.

#### Scenario: Session ID is missing

- **WHEN** an event lacks `SessionID`
- **THEN** the command SHALL use `SourcePath` or `EventID` as a stable fallback key
- **AND** SHALL avoid merging unrelated local usage records.
