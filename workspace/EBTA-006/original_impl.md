# EBTA-006 Original Implementation

## Current Cursor Provider Shape

- `provider/cursor.Provider` implements only `provider.Provider`.
- `CollectSessions(baseDir)` discovers Cursor CSV files, calls `parseUsageCSV(path)`, skips invalid CSV files, and sorts returned rows by `StartTime` then `SessionID`.
- `parseUsageCSV(path)` turns each valid CSV row into one `provider.SessionInfo`.
- Each row uses:
  - `ProviderName = "cursor"`
  - `SessionID = <csv basename>:<rowNumber>`
  - `Title = "<Kind> <Model>"`, falling back to model or `Cursor usage export`
  - `StartTime == EndTime == Date`
  - `Turns = 1`
  - token usage from component columns only

## Compatibility Constraints

- `Date` is parsed with `time.RFC3339Nano`; timestamp offsets are preserved.
- `Kind` is optional, but `Date`, `Model`, and the four token component columns are required.
- Bad rows are skipped.
- Bad CSV files are skipped by the collector loop.
- Blank token cells map to zero.
- `Total Tokens` is intentionally ignored.
- Default source scans `~/.codetok/cursor/*.csv`, plus `imports/` and `synced/`; explicit `baseDir` scans only that directory recursively.

## Gap

Cursor CSV rows already represent timestamped usage records, but the provider does not implement `provider.UsageEventProvider`, so future command event collection would fall back to synthetic session events instead of native Cursor events.
