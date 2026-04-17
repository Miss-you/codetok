# EAP-003 Original Implementation

## Current Flow

- `cmd/daily.go` calls `collectUsageEventsFromProviders` before resolving the daily date window.
- `cmd/session.go` resolves date strings first, but still calls `collectUsageEventsFromProviders` with no range information.
- `cmd/collect.go` iterates filtered providers and calls native `provider.UsageEventProvider.CollectUsageEvents`; legacy providers fall back to one synthetic event per `SessionInfo`.
- `stats.FilterEventsByDateRange` is the first exact date filter for both `daily` and `session`.

## Provider Behavior

- `provider/claude` discovers all `.jsonl` files, then event parsing is currently sequential.
- `provider/codex` discovers all dated `year/month/day/rollout-*.jsonl` files and parses them in parallel.
- `provider/kimi` discovers all session directories with `wire.jsonl` and parses them in parallel.
- `provider/cursor` discovers CSV files and parses all rows into session-like records, then maps rows to usage events.

## Contracts To Preserve

- Reporting commands remain local-only.
- File metadata must only be a candidate filter. Event timestamps remain authoritative through final `stats.FilterEventsByDateRange`.
- `--all` must preserve full-history collection.
- `session` without `--since/--until` may preserve full-history behavior.
- Provider directory overrides remain authoritative.

## Risks

- Skipping by `ModTime` can be wrong if treated as event time.
- Codex dated paths can contain cross-day usage; previous-day files must stay candidates for a selected day.
- Concurrent provider parsing can reorder events, so command behavior must rely on existing aggregation sorting rather than parse order.
