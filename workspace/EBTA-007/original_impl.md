# EBTA-007 Original Implementation

## Current daily path

`runDaily` currently resolves flags and timezone, then still collects `provider.SessionInfo` with `collectSessions(cmd)`. It filters sessions through `stats.FilterByDateRange` and aggregates with `stats.AggregateByDayWithDimension`.

That means daily rows are attributed by `SessionInfo.StartTime`. The selected `--timezone` affects date-window parsing, but not the final date key because session aggregation formats `StartTime` directly.

## Existing event support

The lower layers needed for EBTA-007 already exist:

- `provider.UsageEvent` and `provider.UsageEventProvider`
- `collectUsageEvents(cmd)`, which prefers native events and falls back to synthetic session-start events for legacy providers
- `stats.FilterEventsByDateRange`, which filters inclusive localized date keys
- `stats.AggregateEventsByDayWithDimension`, which groups by `event.Timestamp.In(loc)` and counts distinct sessions per date/group

## Current tests

`cmd/daily_test.go` already covers timezone resolution, date-window parsing, flag conflicts, token units, group-by parsing, JSON output behavior, and dashboard formatting.

The missing command-level coverage is the integration point: `daily` must feed collected usage events into event filtering and event aggregation.

## Likely write set

- `cmd/daily.go`
- `cmd/daily_test.go`
- `openspec/changes/event-based-token-aggregation-daily/*`
- `docs/plans/2026-04-16-event-based-token-aggregation-task.md`
- `workspace/EBTA-007/*`

## Risks

- Formatting a zero `time.Time` bound would create bogus `0001-01-01` filters.
- Filtering events with `stats.FilterByDateRange` would reintroduce timestamp-bound/session-start semantics.
- JSON output must preserve `provider.DailyStats` fields and raw token counts.
- `session` command changes belong to EBTA-008, not this task.
