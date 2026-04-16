## Why

`daily` currently has no shared event-level model, so provider parsers must collapse timestamped token records into session summaries before aggregation. This prevents the stats layer from attributing usage to the calendar day when each token event occurred.

## What Changes

- Add a shared `provider.UsageEvent` model for timestamped token deltas.
- Add an optional `provider.UsageEventProvider` interface while preserving the existing session provider interface.
- Add stats helpers that filter and aggregate usage events by localized event date.
- Preserve existing `DailyStats` JSON field semantics for provider, group, providers, sessions, and token totals.
- No breaking changes in this foundation task.

## Capabilities

### New Capabilities

- `event-based-token-aggregation-core`: Shared event model and stats aggregation foundation for timestamped local token usage events.

### Modified Capabilities

- None.

## Impact

- Affected code: `provider/provider.go`, `stats/events.go`, `stats/events_test.go`.
- No new dependencies.
- No command behavior changes until later task-board items switch callers to event aggregation.
