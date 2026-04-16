## 1. Command Tests

- [x] 1.1 Add failing session command tests proving event-date filtering and filtered totals.
- [x] 1.2 Add failing session command tests proving `--timezone` affects filtering and output dates.
- [x] 1.3 Add failing aggregation tests for provider/session boundaries and fallback keys.
- [x] 1.4 Add focused tests for table output, provider override routing, and invalid dates.

## 2. Session Event Pipeline

- [x] 2.1 Add `--timezone` and a provider-injected session command test seam.
- [x] 2.2 Switch `session` from session collection/filtering to usage-event collection/filtering.
- [x] 2.3 Group filtered events into deterministic session rows.
- [x] 2.4 Render JSON/table dates in the selected timezone while preserving output shape.

## 3. Validation

- [x] 3.1 Run focused session command tests.
- [x] 3.2 Run focused command/stats/provider event tests.
- [x] 3.3 Run repository gates for EBTA-008.
