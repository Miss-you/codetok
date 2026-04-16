## 1. Kimi Event Tests

- [x] 1.1 Add failing tests proving `StatusUpdate` records produce timestamped incremental `UsageEvent` records, including cross-day events and metadata/model fallback behavior.

## 2. Kimi Event Collector

- [x] 2.1 Implement `Provider.CollectUsageEvents` and Kimi event parsing with existing local session discovery semantics.
- [x] 2.2 Preserve existing Kimi session parser behavior.

## 3. Validation

- [x] 3.1 Run focused Kimi event tests.
- [x] 3.2 Run `go test ./provider/kimi`.
