## Context

The shared event model, native-first command collector, and timezone-aware event stats are already in place. `daily` still runs the legacy session pipeline: collect sessions, filter by `SessionInfo.StartTime`, and aggregate by `SessionInfo.StartTime.Format("2006-01-02")`.

## Goals / Non-Goals

**Goals:**

- Make `daily` use `provider.UsageEvent` timestamps for date attribution.
- Reuse the existing `--timezone` and date-window helpers so flag validation stays centralized.
- Preserve JSON field semantics and dashboard rendering behavior.
- Keep reporting local-only.

**Non-Goals:**

- Do not change `session`; EBTA-008 owns session event filtering.
- Do not add new provider parsers; native provider event support is already covered by earlier EBTA tasks.
- Do not change README or e2e fixtures; later EBTA tasks own final docs and acceptance.

## Decisions

- Use `collectUsageEventsFromProviders` inside `daily` instead of `collectSessions`. This preserves provider filters, base directory flags, per-provider directory overrides, missing-directory skip behavior, and legacy provider fallback.
- Keep `resolveDailyDateRange` returning `time.Time` bounds. It already owns mutual exclusions and selected-timezone date parsing, and changing its API would increase churn.
- Add a small command helper that converts non-zero date bounds to `YYYY-MM-DD` keys in the selected location before calling `stats.FilterEventsByDateRange`.
- Use `stats.AggregateEventsByDayWithDimension` for final rows. It already counts distinct sessions per date/group and preserves `provider`, `group_by`, `group`, and `providers` JSON semantics.
- Add a test seam that injects providers and `now` for command-level tests. This avoids relying on the real clock for default rolling-window coverage.

## Risks / Trade-offs

- [Risk] Legacy providers without native events still produce one synthetic event at session start. -> Mitigation: Earlier EBTA tasks migrated Codex, Claude, Kimi, and Cursor; fallback remains compatibility behavior only.
- [Risk] Date-window conversion could reintroduce UTC boundaries. -> Mitigation: Convert bounds with `time.In(loc)` and test Shanghai boundary cases.
- [Risk] JSON grouping fields could drift while switching data sources. -> Mitigation: Add CLI and multi-provider model grouping command tests.
