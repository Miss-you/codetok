## Why

`codetok session` still filters by `provider.SessionInfo.StartTime`, so continued
usage from a session that started before the selected date range is excluded and
in-range sessions include full session totals. Native provider usage events and
timezone-aware event filtering already exist, so the session command can switch
to filtering usage events before grouping them back into session rows.

## What Changes

- Switch `session` collection from session summaries to `provider.UsageEvent` values.
- Add `--timezone` to `session` for date filtering and displayed session dates.
- Filter by localized usage-event dates before grouping provider/session rows.
- Preserve existing JSON field names and table columns.

## Capabilities

### New Capabilities

- `session-event-aggregation`: The `session` command filters local token usage by
  event timestamp and groups matching events by provider/session.

### Modified Capabilities

- None.

## Impact

- Affected code: `cmd/session.go`, `cmd/session_test.go`.
- Depends on existing event collector and stats event-date filtering.
- No new dependencies and no remote provider API calls.
