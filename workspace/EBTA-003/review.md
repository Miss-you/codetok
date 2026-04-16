# EBTA-003 Review Notes

## Planning Review

- Spec compliance review approved `final_impl_v1.md` and `test_strategy.md`.
- Implementation planning review initially requested explicit coverage for cumulative counter reset, first session/title stability, and explicit base-dir precedence over `CODEX_HOME`.
- The planning artifacts were updated and re-reviewed with no remaining must-fix findings.

## Code Review

Two implementation review passes were run.

Must-fix findings from the first pass:

- Mixed `last_token_usage` followed by `total_token_usage` could overcount because last-only events did not advance the cumulative baseline.
- Legacy `parseCodexSession` could drift to a later `session_meta.id` and lose pre-reset usage by taking the last cumulative total.
- `CollectUsageEvents` parsed files serially instead of using bounded parallel parsing like `CollectSessions`.

Fixes:

- Added regression tests for mixed last/total baseline handling.
- Added regression tests for legacy session first metadata and reset aggregation.
- Updated `codexUsageDelta` to advance a synthetic cumulative baseline after last-only usage.
- Updated `parseCodexSession` to accumulate deltas and keep the first session ID.
- Added `provider.ParseUsageEventsParallel` and used it in Codex native event collection.

Re-review result:

- Both reviewers approved the fixes with no remaining must-fix findings.

## Verification

- `go test ./provider ./provider/codex -run 'Test(ParseCodexUsageEvents|ParseCodexSession_KeepsFirstSessionMetadataAndSumsResetUsage|CollectCodexUsageEvents|CodexHome)'`
- `go test ./provider ./provider/codex`
- `make fmt`
- `make test`
- `make vet`
- `make build`
- `make lint`
