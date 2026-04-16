## Context

`daily` already accepts `--timezone` and has helpers for resolving date windows, but its execution path still collects `provider.SessionInfo` and delegates to session-start aggregation. Earlier event aggregation tasks added the shared event model, native provider collectors, command collection bridge, and stats helpers.

## Goals / Non-Goals

**Goals:**

- Make `daily` aggregate by token event date in the selected timezone.
- Make `daily --since/--until` filter by localized event date keys.
- Preserve the existing CLI flags, JSON schema, dashboard formatting, and local-only reporting boundary.

**Non-Goals:**

- Do not change `session`; EBTA-008 owns that command.
- Do not change provider parsers or Cursor sync behavior.
- Do not add e2e fixtures or README updates in this task.

## Decisions

### Reuse the command event bridge

`runDaily` will call `collectUsageEvents(cmd)` instead of `collectSessions(cmd)`. This preserves provider filtering, `--base-dir`, provider-specific directory overrides, native event preference, legacy fallback behavior, and missing-directory handling.

Alternative considered: add a second daily-specific collection path. Rejected because it would duplicate semantics already covered by EBTA-002.

### Convert resolved time bounds to date keys at the command edge

`resolveDailyDateRange` remains the owner of flag constraints and timezone-aware date parsing. `daily` will convert non-zero bounds to `YYYY-MM-DD` keys before calling `stats.FilterEventsByDateRange`.

Alternative considered: change `stats.FilterEventsByDateRange` to accept `time.Time` bounds. Rejected because the stats helper already expresses the desired event-date semantics and has coverage.

### Keep aggregation rules in `stats`

`daily` will delegate grouping and distinct-session counting to `stats.AggregateEventsByDayWithDimension`. The command only wires flags, collection, filtering, output, and dashboard rendering.

Alternative considered: aggregate directly in `cmd/daily.go`. Rejected because it would duplicate provider/group/session semantics and raise drift risk.

## Risks / Trade-offs

- [Risk] Zero date bounds could accidentally become `0001-01-01`. -> Mitigation: add a helper that returns an empty bound for zero times.
- [Risk] Applying timezone only to aggregation but not filtering would keep `--since/--until` wrong. -> Mitigation: command tests cover timezone-aware event filtering.
- [Risk] Legacy fallback providers still use session-start synthetic events. -> Mitigation: all target providers for this plan already have native event collectors; fallback remains migration compatibility.
