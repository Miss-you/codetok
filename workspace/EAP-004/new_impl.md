# EAP-004 Proposed Implementation

## Shape

Add a command-level event iterator and a stats accumulator:

- `stats.NewDailyEventAggregator(dimension, loc)` creates an incremental daily aggregator.
- `DailyEventAggregator.Add(event)` applies the existing daily aggregation semantics to one event.
- `DailyEventAggregator.Results()` returns the same sorted `[]provider.DailyStats` as the materialized path.
- `stats.EventInDateRange` exposes the existing localized date-key predicate for one event.
- `cmd.forEachUsageEventFromProvidersInRange` reuses provider filtering, directory overrides, range-aware dispatch, missing-directory handling, and provider error wrapping while passing events to a consumer callback.
- `cmd.collectUsageEventsFromProvidersInRange` remains as a compatibility adapter for callers that still need a materialized slice.
- `daily` uses the iterator plus aggregator directly, avoiding one all-provider `[]UsageEvent` and the filtered event slice.

## Scope

Files:

- `stats/events.go`
- `stats/events_test.go`
- `cmd/collect.go`
- `cmd/daily.go`
- `cmd/daily_test.go`

No provider parser changes are needed for this task. No OpenSpec change is needed because user-visible behavior, CLI flags, JSON fields, and local-only provider semantics are unchanged.

## Tradeoffs

This is intentionally shallow streaming. It removes command-level materialization in `daily`, but providers still return slices. That is the narrowest useful EAP-004 slice and avoids changing provider parser contracts while EAP-005 can focus on provider-specific parser allocation churn.
