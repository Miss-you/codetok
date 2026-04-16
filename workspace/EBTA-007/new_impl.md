# EBTA-007 Proposed Implementation

## Approach

Switch `daily` to the event pipeline:

1. Keep existing flag resolution and validation order.
2. Collect usage events through `collectUsageEventsFromProviders`.
3. Resolve the same daily date range with `resolveDailyDateRange`.
4. Convert non-zero `since` and `until` bounds into localized `YYYY-MM-DD` date keys.
5. Filter events through `stats.FilterEventsByDateRange`.
6. Aggregate rows through `stats.AggregateEventsByDayWithDimension`.
7. Leave JSON and dashboard rendering unchanged.

## Test Strategy

Add command-level tests with fake native event providers proving:

- Same session events split across event dates.
- `--timezone` changes event date keys.
- Explicit `--since`/`--until` filters by localized event date.
- Default rolling window uses the selected local day boundary.
- CLI grouping JSON fields stay stable.
- Model grouping across providers keeps `group_by`, `group`, and `providers` semantics.

## Scope Control

Do not change:

- Provider parsers
- `session` command
- e2e fixtures
- README docs
- existing stats event semantics
