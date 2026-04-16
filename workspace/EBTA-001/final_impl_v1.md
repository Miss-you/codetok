# EBTA-001 Final Implementation Plan v1

## Implementation

1. Extend `provider/provider.go` with:
   - `UsageEvent`
   - `UsageEventProvider`
2. Create `stats/events.go` with:
   - `AggregateEventsByDayWithDimension`
   - `FilterEventsByDateRange`
   - event-specific group-name and session-key helpers
3. Keep session-based functions unchanged.

## Review

Self-review against task criteria:

- CLI contract impact: low; no command behavior changes yet.
- Provider/stats semantics: aligned with the approved event-first model.
- Go maintainability: small additive API and separate stats file avoid churn.
- Scope control: excludes collector, provider parser, command, and docs work reserved for later tasks.
- Testability: focused stats tests prove split-by-event-day, timezone grouping, distinct sessions, grouping metadata, fallback source keys, and filtering.

Score: 90/100. No high-severity issue found. Subagent review was not used because this session does not have explicit user authorization for subagent dispatch; this plan keeps the review local and evidence-based.
