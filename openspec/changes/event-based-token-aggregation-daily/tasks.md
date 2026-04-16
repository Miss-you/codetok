## 1. Command Tests

- [x] 1.1 Add failing daily command tests for event-date split, timezone date-key filtering, default localized lookback, model grouping JSON semantics, and date-window validation order.

## 2. Daily Event Pipeline

- [x] 2.1 Switch `runDaily` from session collection and session aggregation to usage-event collection, event date filtering, and event aggregation.
- [x] 2.2 Add zero-safe date-bound conversion helpers owned by `cmd/daily.go`.

## 3. Validation

- [x] 3.1 Run focused daily command tests and relevant stats event tests.
- [x] 3.2 Run repository gates required by EBTA-007.
