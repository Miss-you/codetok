# EBTA-001 Original Implementation Notes

## Current State

- Shared provider data is centered on `provider.SessionInfo`.
- `stats.AggregateByDayWithDimension` groups by `SessionInfo.StartTime.Format("2006-01-02")`.
- `stats.FilterByDateRange` filters by `SessionInfo.StartTime` using `time.Time` bounds.
- `DailyStats.Sessions` currently increments once per aggregated `SessionInfo`.
- There is no shared `UsageEvent` type, no optional event provider interface, and no event-level daily aggregation.

## Consequence

The current foundation cannot represent multiple token events from the same session across different calendar days. Provider parsers must collapse timestamped log records into a single `SessionInfo` before `stats` sees them.
