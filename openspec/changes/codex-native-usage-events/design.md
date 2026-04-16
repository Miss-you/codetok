## Context

The shared `provider.UsageEvent` model and event stats helpers already exist from `event-based-token-aggregation-core`. Codex is still session-based: `CollectSessions` scans rollout JSONL files and keeps the latest cumulative `total_token_usage` as one `SessionInfo` per file.

EBTA-003 migrates only the Codex provider producer side. Command callers continue to use session aggregation until later tasks switch them to event collection.

## Goals / Non-Goals

**Goals:**

- Emit native Codex `UsageEvent` records from local `token_count` lines.
- Preserve existing Codex session parser behavior and tests.
- Share Codex source directory resolution between session and event collection.
- Honor `$CODEX_HOME/sessions` when no explicit Codex directory is provided.
- Keep implementation local-only with no provider API calls.

**Non-Goals:**

- Do not change `daily` or `session` command aggregation.
- Do not migrate Claude, Kimi, or Cursor providers.
- Do not change `TokenUsage` output field semantics.
- Do not add cost or remote sync behavior.

## Decisions

### Add Codex Event Parser Beside Session Parser

Add `CollectUsageEvents` and `parseCodexUsageEvents` beside `CollectSessions` and `parseCodexSession`. This keeps existing callers stable and lets later command tasks opt into native events through `UsageEventProvider`.

Alternative considered: replace `parseCodexSession` with event parsing plus event-to-session conversion. Rejected for EBTA-003 because command behavior has not migrated yet and existing session parser compatibility is part of the task.

### Shared Codex Source Discovery

Factor Codex root resolution and date-directory JSONL discovery into helpers used by both collectors. The resolution order is explicit `baseDir`, `$CODEX_HOME/sessions`, then `~/.codex/sessions`.

Alternative considered: apply `CODEX_HOME` only to `CollectUsageEvents`. Rejected because users expect the Codex provider default root to be consistent across local reporting paths.

### File-Local Delta Recovery

For each rollout file, prefer `last_token_usage` as an already-incremental event. When only `total_token_usage` is present, subtract the previous cumulative total in the same file. If any cumulative counter decreases, treat the current total as a reset and emit it as a fresh delta rather than producing negative usage.

Alternative considered: skip reset records. Rejected because that would silently lose post-reset usage.

### Stable Metadata

Use the first `session_meta.id` and first user message title as stable file metadata for emitted events. Track the latest valid model context so token records without model metadata still carry the active Codex model.

Alternative considered: update session ID or title whenever later metadata appears. Rejected because it can split one rollout file across identities and break distinct session counting.

## Risks / Trade-offs

- [Risk] Codex may introduce new token usage field names. -> Mitigation: keep parsing tolerant and skip malformed records; future tasks can add schema variants with focused tests.
- [Risk] Reset handling may overcount if a lower cumulative counter does not represent a real reset. -> Mitigation: avoid negative usage and document this conservative local-file behavior in tests.
- [Risk] Absolute `SourcePath` can vary between machines. -> Mitigation: it is metadata and a fallback session key only; `session_meta.id` remains preferred.
