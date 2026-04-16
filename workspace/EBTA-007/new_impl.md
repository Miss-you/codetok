# EBTA-007 Proposed Implementation

## Goal

Switch only the `daily` command from session-start aggregation to usage-event aggregation.

## Implementation

In `runDaily`, keep the existing flag parsing and validation, but replace the session pipeline:

1. Call `collectUsageEvents(cmd)`.
2. Resolve the daily date window with `resolveDailyDateRange`.
3. Convert non-zero `since` and `until` bounds to localized `YYYY-MM-DD` date keys.
4. Filter events with `stats.FilterEventsByDateRange`.
5. Aggregate with `stats.AggregateEventsByDayWithDimension(events, groupBy, loc)`.

Add small command-owned helpers:

- `buildDailyStatsFromUsageEvents(events, since, until, loc, groupBy)`
- `dailyDateBound(t, loc)`

`dailyDateBound` must return `""` for zero times and normalize nil locations to `time.Local`.

## Preserved behavior

- `--all`, `--days`, `--since`, and `--until` constraints stay in `resolveDailyDateRange`.
- `--timezone` remains command-owned and defaults to `time.Local`.
- JSON output continues to encode `[]provider.DailyStats`.
- Dashboard `--unit` and `--top` behavior remains unchanged.
- Provider filtering and directory overrides continue through `collectUsageEvents`.

## Out of scope

- `session` event filtering
- e2e fixtures
- README updates
- provider parser changes
