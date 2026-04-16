## Context

The shared event model, native-first event collector, provider event parsers, and
daily command event aggregation already exist. `session` remains on the legacy
session summary path.

## Goals / Non-Goals

**Goals:**

- Make `session` use `provider.UsageEvent.Timestamp` for date filtering.
- Include sessions with in-range events even when they started earlier.
- Sum only filtered events in each session row.
- Render session dates in the selected timezone.
- Preserve existing JSON field names and table columns.

**Non-Goals:**

- Do not change provider parsers.
- Do not add e2e cross-day fixtures; EBTA-009 owns that acceptance layer.
- Do not update README docs; EBTA-010 owns final user-facing docs.
- Do not remove `provider.SessionInfo` or legacy collection helpers.

## Decisions

- Reuse `collectUsageEventsFromProviders` so provider filters, directory overrides,
  missing-directory behavior, and legacy fallback remain centralized.
- Keep session grouping in `cmd/session.go` because it shapes command output rather
  than reusable stats.
- Group by provider plus `SessionID`, then `SourcePath`, then `EventID`, with a
  provider-local anonymous fallback.
- Keep JSON schema stable. `ModelName` is preserved internally but not emitted as
  a new JSON field in this task.

## Risks / Trade-offs

- [Risk] `Turns` cannot be exactly reconstructed from usage events. -> Mitigation:
  use included usage-event count as best-effort and keep the field stable.
- [Risk] Events without session IDs could be merged accidentally. -> Mitigation:
  use source/event fallback keys and focused tests.
