# EAP-004 Final Implementation Draft

## Decision

Implement command-level direct-to-aggregator daily collection. Do not introduce a provider-level streaming interface in this task.

## Implementation

1. Refactor `stats.AggregateEventsByDayWithDimension` around an exported incremental `DailyEventAggregator`.
2. Add `stats.EventInDateRange` so `daily` can keep exact event-date filtering without allocating a filtered slice.
3. Add `forEachUsageEventFromProvidersInRange` in `cmd/collect.go`.
4. Keep `collectUsageEventsFromProvidersInRange` as an adapter over the iterator for existing session/tests compatibility.
5. Change `runDailyWithProviders` to call `aggregateDailyUsageEventsFromProvidersInRange`.

## Compatibility

- JSON output remains raw `provider.DailyStats`.
- Dashboard output still renders from `[]provider.DailyStats`.
- Provider filtering and per-provider directory overrides are unchanged.
- Missing local provider roots are still skipped.
- Provider errors keep existing contextual wrapping.
- Exact event timestamp filtering remains authoritative after candidate collection.

## Validation

Focused RED tests:

- `TestDailyEventAggregator_MatchesMaterializedAggregation`
- `TestAggregateDailyUsageEventsFromProvidersInRange_MatchesMaterializedPath`

Additional protection:

- Provider error context test for direct aggregation.
- JSON parity test against materialized filter-and-aggregate.
- Dashboard parity test against `printDailyDashboard` with materialized stats.
