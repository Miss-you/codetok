## 1. Command Tests

- [x] 1.1 Add failing daily command tests proving same-session usage events split across event dates.
- [x] 1.2 Add failing daily command tests proving `--timezone`, explicit date filters, and default rolling windows use localized event dates.
- [x] 1.3 Add failing daily command tests proving CLI and model JSON grouping fields remain stable with usage events.

## 2. Daily Event Pipeline

- [x] 2.1 Add a provider/clock seam for deterministic daily command tests.
- [x] 2.2 Switch `daily` from session collection/filtering/aggregation to usage event collection/filtering/aggregation.
- [x] 2.3 Convert resolved date bounds into localized date keys for event filtering.

## 3. Validation

- [x] 3.1 Run focused daily command tests.
- [x] 3.2 Run focused command/stats event tests.
- [x] 3.3 Run repository gates for EBTA-007.
