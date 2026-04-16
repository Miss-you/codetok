# EBTA-002 Original Implementation

## Current Command Collection

- `cmd/collect.go` only exposes `collectSessions` and `collectSessionsFromProviders`.
- The helper filters providers with `--provider`, chooses `--base-dir` or the per-provider `--<name>-dir`, calls `CollectSessions`, skips missing directories, and wraps provider errors.
- There is no command helper that prefers `provider.UsageEventProvider.CollectUsageEvents`.
- Legacy providers can still compile because `provider.Provider` remains session-based.

## Current Daily Helpers

- `cmd/daily.go` has no `--timezone` flag or reusable timezone resolver.
- `runDaily` still calls `collectSessions`, then `stats.FilterByDateRange`, then `stats.AggregateByDayWithDimension`.
- `resolveDailyDateRange` parses explicit dates with `time.Parse`, so bounds are UTC.
- The default rolling window is also anchored to `time.Now().UTC()`, not local day boundaries.

## Current Tests

- `cmd/collect_test.go` covers session collection, provider filtering, directory overrides, missing directories, and wrapped provider errors.
- `cmd/daily_test.go` covers UTC date ranges, flag conflicts, token units, grouping, `--top`, and dashboard formatting.
- There are no `cmd` tests for native event collection, fallback event synthesis, timezone resolution, or local-day default windows.

## Scope Boundary

EBTA-001 already added `provider.UsageEvent`, `provider.UsageEventProvider`, and `stats` event aggregation helpers. EBTA-002 should bridge commands to that foundation but must not switch `daily` or `session` reporting to event aggregation; that belongs to EBTA-007 and EBTA-008.
