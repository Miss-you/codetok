# EBTA-001 Candidate Implementation

## Approach

- Add `provider.UsageEvent` with the exact shared fields from the approved design.
- Add optional `provider.UsageEventProvider` while keeping the existing `Provider` interface intact.
- Add `stats/events.go` instead of modifying session aggregation, so legacy command behavior remains untouched until later tasks switch callers.
- Implement `AggregateEventsByDayWithDimension` by mirroring the existing `DailyStats` grouping metadata while using event timestamps in a supplied timezone.
- Count sessions with a distinct session/container key per date/group, preferring `SessionID` and falling back to `SourcePath`.
- Implement `FilterEventsByDateRange` using inclusive `YYYY-MM-DD` date keys in the selected timezone.

## Boundaries

- No provider parser changes in this task.
- No `cmd` collection or flag changes in this task.
- No JSON schema change beyond introducing the new Go type for future use.
