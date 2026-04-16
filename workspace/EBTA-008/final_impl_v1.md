# EBTA-008 Final Implementation V1

## Scope

Switch only the `session` command from session-start filtering to usage-event filtering.
Do not change providers, daily behavior, README docs, or e2e fixtures in this task.

## Implementation Steps

1. Add `--timezone` to `session` with the same validation as `daily`.
2. Add `runSessionWithProviders(cmd, args, providers)` and keep `runSession` as the
   registry-backed Cobra entry point.
3. Add `resolveSessionEventFilterDates` to validate `--since/--until` and return
   inclusive date keys.
4. Replace the `collectSessions` + `stats.FilterByDateRange` path with:
   `collectUsageEventsFromProviders`, `stats.FilterEventsByDateRange`, and
   `aggregateSessionEvents`.
5. Add localized date rendering for JSON/table output without changing field names
   or table columns.

## Acceptance Checklist

- `session --since/--until` includes sessions that have matching in-range usage events,
  even when the session began earlier.
- Session token totals include only filtered events.
- `session --timezone Asia/Shanghai` changes filtering and displayed date by Shanghai
  calendar day.
- Same session IDs across different providers remain separate.
- Existing provider directory overrides and provider filtering still flow through the
  existing event collector helper.
- Focused RED tests fail before implementation and pass after implementation.
