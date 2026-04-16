## Why

`codetok daily` still filters and groups by `SessionInfo.StartTime`, so token usage from sessions that continue across midnight can be attributed to the wrong day or excluded from the selected window. Native provider usage events and timezone-aware event stats already exist, so the daily command can now switch to event timestamps.

## What Changes

- Switch `daily` collection from session summaries to `provider.UsageEvent` values through the native-first collector bridge.
- Filter daily rows by each event's localized calendar date using `--timezone`, `--since`, `--until`, `--days`, and `--all`.
- Aggregate daily rows with event-date stats while preserving existing JSON fields and CLI/model grouping semantics.
- Preserve existing daily flag constraints and dashboard-only flag behavior.

## Capabilities

### New Capabilities

- `daily-event-aggregation`: The `daily` command aggregates local token usage by event timestamp and selected timezone.

### Modified Capabilities

- None.

## Impact

- Affected code: `cmd/daily.go`, `cmd/daily_test.go`.
- Depends on existing command collection helpers and stats event aggregation.
- No new dependencies and no remote provider API calls.
