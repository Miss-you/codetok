# EAP-007 Original Implementation

## Current State

`codetok daily` is already using the optimized event collection path:

1. `cmd/daily.go` resolves timezone and date bounds before collection.
2. `cmd/collect.go` dispatches to `collectUsageEventsFromProvidersInRange`.
3. Providers that implement `provider.RangeAwareUsageEventProvider` receive `provider.UsageEventCollectOptions`.
4. `stats.FilterEventsByDateRange` remains the authoritative event timestamp filter.
5. `stats.AggregateEventsByDayWithDimension` performs the final day/group aggregation.

This preserves event-timestamp attribution while letting providers skip provably inactive local files. After rebasing onto `origin/main`, EAP-004 has also landed the streaming daily path, so final acceptance must include that evidence instead of treating streaming aggregation as deferred.

## Existing Guardrails

The current tree has focused tests around the performance-sensitive boundaries:

- `cmd/daily_test.go` covers date resolution, `--all`, range-aware collection, timezone filtering, and final event filtering.
- `cmd/session_test.go` covers range-aware event collection for explicit session date filters and full-history collection when no date range is present.
- `cmd/collect_test.go` covers range-aware dispatch, full-history fallback, directory overrides, and error wrapping.
- `provider/claude`, `provider/codex`, `provider/kimi`, and `provider/cursor` tests cover candidate filtering, in-window event retention, metrics, and provider-specific edge cases.
- `stats/events_test.go` covers localized date filtering and cross-day event aggregation semantics.
- `e2e/e2e_test.go` includes cross-day event acceptance checks against the built binary.

## Baseline Evidence

The design plan records the pre-optimization local timing:

- `./bin/codetok daily`: 5.26s wall on first run
- `./bin/codetok daily`: 5.07s wall on second run
- direct profile harness, all providers: 5.93s total
- collection time in that harness: 5.92s
- filter plus aggregation in that harness: under 10ms

That evidence showed provider parsing dominated runtime. EAP-002, EAP-003, and EAP-005 addressed the parser and candidate filtering surfaces already.

## Current Risks

- Local wall-clock timing is machine and dataset dependent, so EAP-007 should record it as acceptance evidence, not a portable test contract.
- EAP-004 is done on `origin/main`; EAP-007 must include it in dependency and evidence accounting.
- EAP-006 is done on `origin/main`; EAP-007 must include it in dependency and evidence accounting.
- No reporting command should call provider APIs during acceptance; manual smoke checks must use the built local binary only.
