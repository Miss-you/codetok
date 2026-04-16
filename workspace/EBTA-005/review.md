# Review Notes

Review agents:

- Spec compliance review found one low-severity drift: `UsageEvent.EventID` was described in the design but was not populated.
- Code quality review found no must-fix issue and noted the same `EventID` gap as non-blocking for current stats.

Resolution:

- Added failing EventID assertions.
- Implemented stable Kimi event IDs using `message_id` when present and source line number otherwise.
- Added a test proving wire payload model fallback wins over log fallback when metadata has no model.

Post-review verification:

- `go test ./provider/kimi -run 'TestParseKimiUsageEvents_StatusUpdatesEmitIncrementalEvents|TestCollectKimiUsageEvents_WireModelWinsOverLogFallback'`
- `go test ./provider/kimi`
- `make fmt`
- `make test`
- `make vet`
- `make build`
- `make lint`

Remaining risk:

- Kimi `StatusUpdate.token_usage` remains intentionally treated as incremental. A future real-log fixture should drive a separate task if Kimi starts emitting cumulative counters.
