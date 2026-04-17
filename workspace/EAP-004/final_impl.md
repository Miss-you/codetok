# EAP-004 Final Implementation

## Accepted Approach

EAP-004 uses a shallow streaming path for `daily`: events are consumed provider-by-provider and added directly to a stats accumulator. This removes the command-level all-provider event slice and the separate filtered event slice while preserving provider APIs.

OpenSpec change: not used. This task preserves user-facing behavior and contracts; it only changes the internal daily aggregation path.

## Files Changed

- `stats/events.go`: added `DailyEventAggregator` and `EventInDateRange`; refactored materialized aggregation through the same accumulator.
- `cmd/collect.go`: added a callback-based provider event iterator and kept the materialized collector as an adapter.
- `cmd/daily.go`: changed `daily` to aggregate provider events directly.
- `stats/events_test.go`: added accumulator/materialized parity coverage.
- `cmd/daily_test.go`: added direct aggregation, provider error context, JSON parity, and dashboard parity coverage.

## Expected Behavior

`codetok daily` output remains identical to the materialized path for JSON and dashboard modes. Parser/provider operational errors still carry provider context. Reporting remains local-only.
