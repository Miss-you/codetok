# EBTA-008 New Implementation

## Approach

Keep the change command-local:

- add `--timezone` to `session`;
- make `runSession` delegate to `runSessionWithProviders` for test injection;
- collect `provider.UsageEvent` values with `collectUsageEventsFromProviders`;
- validate `--since` and `--until` as date strings in the selected timezone;
- filter events with `stats.FilterEventsByDateRange`;
- aggregate filtered events into `provider.SessionInfo` rows for the existing output
  renderer.

## Session Event Aggregation

Group by provider plus stable session key:

1. `SessionID` when present;
2. `SourcePath` when `SessionID` is empty;
3. `EventID` when both are empty;
4. a provider-local anonymous fallback.

For each group, sum token fields, preserve first non-empty metadata, track earliest
included event as `StartTime`, latest included event as `EndTime`, and keep `Turns`
as the number of included usage events. Sort rows by `StartTime`, provider, and
session ID for deterministic output.

## Output Contract

Keep the existing JSON field names and table columns. Do not add `model` to session
JSON in this task; preserve `ModelName` internally in the aggregated `SessionInfo`.
Render displayed `date` from the first included event timestamp in the selected
timezone.
