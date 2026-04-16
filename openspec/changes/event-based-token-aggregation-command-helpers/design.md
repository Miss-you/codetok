## Context

`EBTA-001` added `provider.UsageEvent` and stats-level event aggregation, but command code still only collects `provider.SessionInfo`. Provider event implementations will arrive incrementally, so command code needs an additive bridge that supports native events without breaking legacy providers.

`daily` also needs reusable timezone/date-window resolution before later tasks can filter event dates correctly. EBTA-002 adds the helper surface and validation, while leaving the current session aggregation path intact.

## Decisions

### Native Events First, Session Fallback Second

`collectUsageEventsFromProviders` uses the same provider filtering and directory override rules as `collectSessionsFromProviders`. For providers implementing `provider.UsageEventProvider`, it calls `CollectUsageEvents` and returns those events unchanged.

For legacy providers, it calls `CollectSessions` and emits one fallback event per session using `SessionInfo.StartTime` as the event timestamp. This keeps Cursor and future providers compiling during migration.

### Preserve Local-Only Semantics

Missing local data directories are skipped with `os.IsNotExist`, matching session collection. Other provider errors are wrapped with provider context and returned.

### Timezone Resolution Is Command-Owned

`resolveTimezone` belongs in `cmd` because it validates a user-facing flag. Empty input resolves to `time.Local`; named input uses `time.LoadLocation`; invalid input returns an `invalid --timezone` error.

### Date Windows Use the Selected Location

`resolveDailyDateRange` parses explicit dates with `time.ParseInLocation` and computes default `--days` windows from midnight in `now.In(loc)`. This establishes local calendar semantics without changing the current session aggregation call site.

## Risks / Trade-offs

- Fallback events retain session-start timestamp semantics. This is intentional compatibility scaffolding until native provider event collectors land.
- `daily --timezone` validates now, but full event-date grouping waits for EBTA-007.
