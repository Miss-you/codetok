## Context

`codetok` currently exposes `provider.SessionInfo` as the shared provider output. The stats package groups daily rows by `SessionInfo.StartTime`, which is not precise enough for logs where a long-running session emits token usage across multiple days. `EBTA-001` establishes the shared event-level model and stats helpers that later provider and command tasks can consume.

## Goals / Non-Goals

**Goals:**

- Add an additive provider-level event model for timestamped token deltas.
- Add timezone-aware event filtering and aggregation in `stats`.
- Preserve existing session aggregation behavior until later tasks switch command callers.
- Keep `DailyStats` grouping metadata compatible with current JSON semantics.

**Non-Goals:**

- Do not implement native event parsing for Codex, Claude, Kimi, or Cursor.
- Do not switch `daily` or `session` command behavior.
- Do not remove or rename `SessionInfo`.

## Decisions

### Additive `UsageEvent` API

Add `UsageEvent` and `UsageEventProvider` beside existing provider types. This avoids breaking current provider implementations and lets later tasks migrate providers independently.

Alternative considered: replace `CollectSessions` directly. Rejected because it would force all providers and commands to migrate in one unsafe step.

### Separate Event Stats File

Create `stats/events.go` instead of modifying `stats/aggregator.go` heavily. Existing session aggregation remains stable, while event aggregation can be tested independently.

Alternative considered: convert events to synthetic sessions and reuse `AggregateByDayWithDimension`. Rejected because session count semantics would either count events or require hidden synthetic state.

### Session Count Key

Count distinct provider/session keys per date/group, preferring `SessionID` and falling back to `SourcePath`. `EventID` is intentionally excluded so logs without session IDs do not inflate session counts to event counts.

Alternative considered: count every event. Rejected because the approved design defines `DailyStats.Sessions` as contributing session count, not token event count.

## Risks / Trade-offs

- [Risk] Events missing both `SessionID` and `SourcePath` collapse into one anonymous contributing session per date/group. → Mitigation: provider tasks should populate at least one stable container identifier.
- [Risk] Later command integration may need date parsing errors surfaced to users. → Mitigation: `EBTA-002` owns command helper and timezone/date-window error handling.
