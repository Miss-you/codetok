# EAP-003 Test Strategy

## Prove Before Implementing

1. Cmd collector:
   - range-aware providers receive `UsageEventCollectOptions`
   - no range uses legacy `CollectUsageEvents`
   - legacy session fallback still works
   - per-provider directory overrides win over `--base-dir`
   - `--provider` filtering applies before range-aware collection

2. Daily command:
   - default 7-day window is resolved before provider collection
   - `--all` bypasses range-aware collection
   - date flag mutual exclusion behavior is unchanged
   - invalid date flags fail before provider collection; this is intentional so invalid user input does not spend time parsing provider files

3. Session command:
   - `--since/--until` uses range-aware collection
   - `--until` is expanded to the full local day before provider options are passed
   - timezone boundary events are preserved by provider-side timestamp helpers
   - no date range preserves full-history collection

4. Providers:
   - Claude/Kimi skip files whose relevant JSONL `ModTime` is safely before `Since`
   - Claude/Kimi include cross-day sessions whose file was modified after `Until`
   - Codex includes previous-day dated files for cross-day events even when `ModTime` is old
   - Codex skips older dated files only when path date is outside the one-day lookback and `ModTime < Since`
   - Cursor preserves explicit directory discovery and returns only in-range row events for range-aware calls
   - Provider metrics prove considered, skipped, parsed, and emitted counts on synthetic fixtures

## Verification Commands

- Focused red/green checks while implementing:
  - `go test ./cmd -run 'TestCollectUsageEventsFromProviders|TestRunDaily|TestRunSession'`
  - `go test ./provider/claude ./provider/codex ./provider/kimi ./provider/cursor`
- Final gates:
  - `make fmt`
  - `make test`
  - `make vet`
  - `make lint` if `golangci-lint` exists
  - `make build`
  - `go test -count=1 ./e2e -run TestEventBasedCrossDayAcceptance`
