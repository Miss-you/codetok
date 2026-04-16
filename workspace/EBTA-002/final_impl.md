# EBTA-002 Final Implementation

## Chosen Design

Add command-side helpers that let later commands collect `provider.UsageEvent` values without forcing every provider to migrate at once. Add reusable timezone/date-window helpers so later daily/session integration can use local calendar dates consistently.

## Files

- `cmd/collect.go`: add `collectUsageEvents` and `collectUsageEventsFromProviders`.
- `cmd/collect_test.go`: test native event collection and legacy session fallback.
- `cmd/daily.go`: add `--timezone`, `resolveTimezone`, and location-aware date-window resolution.
- `cmd/daily_test.go`: test timezone resolution, invalid timezone errors, and local-day default windows.

## Collector Rules

- Use the same `--provider`, `--base-dir`, and `--<provider>-dir` semantics as session collection.
- If a provider implements `provider.UsageEventProvider`, call `CollectUsageEvents` and keep returned events unchanged.
- Otherwise call `CollectSessions` and emit one fallback event per session using `StartTime` as `Timestamp`.
- Skip missing local data directories and wrap all other provider errors with context.

## Timezone Rules

- Empty `--timezone` resolves to `time.Local`.
- Named timezones must be valid IANA names.
- Date parsing and default `--days` windows use the resolved location, including local midnight boundaries.

## OpenSpec Change

- Change: `event-based-token-aggregation-command-helpers`.
- Scope: command-layer event collection bridge and daily timezone/date-window helpers.

## Explicit Boundaries

- EBTA-002 does not switch `daily` or `session` to event aggregation; EBTA-007 and EBTA-008 own those behavior changes.
