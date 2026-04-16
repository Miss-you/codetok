# EBTA-001 Final Implementation Plan

## Chosen Design

Add the event model and event stats foundation without switching any command callers yet.

## Files

- `provider/provider.go`: add `UsageEvent` and `UsageEventProvider`.
- `stats/events.go`: add event filtering and aggregation helpers.
- `stats/events_test.go`: add focused tests for the new behavior.

## Semantics

- Event dates use `event.Timestamp.In(loc).Format("2006-01-02")`.
- `nil` timezone resolves to `time.Local`.
- Date filtering is inclusive on local date keys.
- Grouping supports the existing `cli` and `model` dimensions.
- `DailyStats.Sessions` counts distinct provider/session keys per date/group, using `SessionID` first and `SourcePath` second.
- `EventID` is not part of the session-count fallback.

## Out of Scope

- `cmd/collect.go` bridge helpers.
- `daily` and `session` command behavior changes.
- Native provider event parsers.
- README and e2e updates.
