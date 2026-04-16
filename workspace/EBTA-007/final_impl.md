# EBTA-007 Final Implementation

## Approved Scope

Switch only `codetok daily` to event-date aggregation. Leave `session`, e2e fixtures, provider parsers, and README docs for their separate task-board items.

## Implementation

1. Add command tests first in `cmd/daily_test.go`.
   - Use an isolated fake `provider.UsageEventProvider` registered under a unique name and selected through `--provider`.
   - Make the fake provider's `CollectSessions` return session-start data that would produce the wrong result if `runDaily` still used sessions.
   - Use helper-level fixed-clock tests for default `--days` local-midnight behavior.
   - Prefix command-level daily tests with `TestDaily` so the task-board gate `go test ./cmd -run TestDaily` exercises the new behavior.
2. In `runDaily`, resolve group, timezone, and daily date window before provider collection.
3. Replace the collection/aggregation path with:
   - `collectUsageEvents(cmd)`
   - zero-safe conversion of date bounds to localized date keys
   - `stats.FilterEventsByDateRange`
   - `stats.AggregateEventsByDayWithDimension`
4. Preserve dashboard and JSON behavior exactly.

## OpenSpec

Use change `event-based-token-aggregation-daily`.

## Review Result

Artifact review passed at 93/100 with no blocking issues. The implementation must carry forward the deterministic fake provider strategy and fixed-clock default-window helper test.
