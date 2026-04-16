# EBTA-007 Original Implementation

## Current Flow

`cmd/daily.go` currently wires `runDaily` through the legacy session path:

1. Resolve flags and grouping.
2. Resolve `--timezone`.
3. Collect `provider.SessionInfo` values with `collectSessions`.
4. Resolve the date range with `resolveDailyDateRange`.
5. Filter with `stats.FilterByDateRange`.
6. Aggregate with `stats.AggregateByDayWithDimension`.
7. Render the same `[]provider.DailyStats` as JSON or dashboard output.

The gap is that both filtering and aggregation are based on `SessionInfo.StartTime`.

## Existing Event Support

The event path already exists:

- `provider.UsageEvent` and `provider.UsageEventProvider`
- `cmd.collectUsageEventsFromProviders`
- `stats.FilterEventsByDateRange`
- `stats.AggregateEventsByDayWithDimension`

The collector bridge already preserves provider filtering, `--base-dir`, provider-specific directory overrides, missing-directory skip behavior, native event collection, and legacy fallback.

## Touch Points

Primary implementation target:

- `cmd/daily.go`

Primary tests:

- `cmd/daily_test.go`

No provider parser or stats implementation changes are needed for EBTA-007.

## Caveats

- `resolveDailyDateRange` returns `time.Time` bounds, while event filtering takes date strings.
- JSON `nil` output must continue to normalize to `[]`.
- Dashboard-only validation behavior must stay unchanged: JSON ignores invalid `--unit` and `--top`.
