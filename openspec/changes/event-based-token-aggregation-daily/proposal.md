## Why

`codetok daily` still aggregates by session start time even though providers now expose timestamped usage events. This keeps cross-day usage attached to the wrong local date and makes `--timezone` ineffective for daily grouping.

## What Changes

- Switch `daily` to collect usage events through the command event bridge.
- Filter daily input by each event's localized date key.
- Aggregate daily rows with event timestamps in the selected timezone.
- Preserve existing daily flag constraints, dashboard-only unit/top behavior, and JSON field semantics.

## Capabilities

### New Capabilities

- `event-based-token-aggregation-daily`: Daily command behavior for event-date token aggregation and timezone-aware filtering.

### Modified Capabilities

- None.

## Impact

- Affected code: `cmd/daily.go`, `cmd/daily_test.go`.
- Reuses existing `provider.UsageEvent`, command collection, and `stats` event helpers.
- No new dependencies.
- No remote provider API calls.
- No `session` command behavior change.
