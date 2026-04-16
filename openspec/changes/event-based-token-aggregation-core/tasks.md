## 1. Event Model

- [x] 1.1 Add `UsageEvent` and `UsageEventProvider` to `provider/provider.go`.

## 2. Event Stats

- [x] 2.1 Add failing tests for cross-day event aggregation, timezone grouping, session counting, source-path fallback, model grouping, and date filtering.
- [x] 2.2 Implement `AggregateEventsByDayWithDimension` and `FilterEventsByDateRange`.

## 3. Validation

- [x] 3.1 Run focused `stats` tests and package tests for `provider` and `stats`.
- [x] 3.2 Run repository gates required by the task workflow.
