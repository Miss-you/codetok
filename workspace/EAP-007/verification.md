# EAP-007 Verification

Verified in `.worktrees/eap-007-final-acceptance` on `2026-04-17 16:54:55 CST` after rebasing onto `origin/main` with EAP-004 and EAP-006.

## Focused Correctness Checks

- `go test ./cmd -run 'Test(CollectUsageEventsFromProviders|RunDaily|RunSession|ResolveDailyDateRange|ResolveSession|AggregateDailyUsageEventsFromProvidersInRange)' -count=1` passed (`ok github.com/miss-you/codetok/cmd 2.099s`).
- `go test ./provider/claude ./provider/codex ./provider/kimi ./provider/cursor -run 'Test.*UsageEventsInRange|TestCollectClaudeUsageEventsWithParser_ParsesInParallelAndSorts|TestParseCodexUsageEvents' -count=1` passed:
  - `provider/claude` 1.854s
  - `provider/codex` 1.624s
  - `provider/kimi` 0.822s
  - `provider/cursor` 1.108s
- `go test ./stats -run 'Test(FilterEventsByDateRange|AggregateEventsByDayWithDimension|DailyEventAggregator)' -count=1` passed (`ok github.com/miss-you/codetok/stats 1.362s`).
- `go test ./e2e -run TestEventBasedCrossDayAcceptance -count=1` passed (`ok github.com/miss-you/codetok/e2e 10.960s`).

## Metric Evidence

The focused provider tests assert `UsageEventCollectMetrics` on synthetic fixtures:

- Claude inactive skip: considered=2 skipped=1 parsed=1 emitted=1.
- Claude cross-day keep: considered=1 skipped=0 parsed=1 emitted=2.
- Codex inactive skip: considered=2 skipped=1 parsed=1 emitted=1.
- Codex previous-day keep: considered=1 skipped=0 parsed=1 emitted=2.
- Codex older-file/recent-mtime keep: considered=1 skipped=0 parsed=1 emitted=1.
- Codex unparseable dated path keep: considered=1 skipped=0 parsed=1 emitted=1.
- Kimi inactive skip: considered=2 skipped=1 parsed=1 emitted=1.
- Kimi modified-after-until keep: considered=1 skipped=0 parsed=1 emitted=1.
- Cursor row filter: considered=1 skipped=0 parsed=1 emitted=1.
- Cursor parse-attempt count: considered=2 skipped=0 parsed=2 emitted=1.

## Benchmarks

- `go test ./provider/claude -run '^$' -bench 'BenchmarkCollectClaudeUsageEventsSynthetic$' -count=1` passed:
  - `BenchmarkCollectClaudeUsageEventsSynthetic-10 146 8753133 ns/op 600.0 events/op 100.0 files/op 105874024 B/op 13406 allocs/op`
- `go test ./provider/codex -run '^$' -bench 'BenchmarkParseCodexUsageEventsSynthetic$' -count=1` passed:
  - `BenchmarkParseCodexUsageEventsSynthetic-10 714 1858328 ns/op 38.87 MB/s 1720805 B/op 9716 allocs/op`
- `go test ./cmd -run '^$' -bench 'BenchmarkDailyAggregationMaterializedVsStreaming$' -benchmem -count=1` passed:
  - `materialized-10 93 10955695 ns/op 25470659 B/op 100081 allocs/op`
  - `streaming-10 132 8407405 ns/op 2410067 B/op 100076 allocs/op`

## Full Gates

- `make fmt` passed (`go fmt ./...`).
- `go clean -testcache` then `make test` passed with uncached package execution:
  - `cmd` 1.344s, coverage 83.4%
  - `cursor` 2.415s, coverage 72.3%
  - `e2e` 230.670s
  - `provider` 1.648s, coverage 78.2%
  - `provider/claude` 2.845s, coverage 83.0%
  - `provider/codex` 2.664s, coverage 82.4%
  - `provider/cursor` 3.368s, coverage 89.7%
  - `provider/kimi` 3.079s, coverage 84.7%
  - `stats` 1.879s, coverage 85.2%
- `make vet` passed (`go vet ./...`).
- `make lint` passed (`0 issues.`).
- `make build` passed and wrote `bin/codetok`.

## Built-Binary Smoke And Timing

All smoke checks used `./bin/codetok` after `make build`; command output was redirected to `/tmp/eap007_*.txt`.

| Command | Result | Evidence |
| --- | ---: | --- |
| `./bin/codetok daily` run 1 | 0.48s real | dashboard output, 863 bytes |
| `./bin/codetok daily` run 2 | 0.51s real | dashboard output, 863 bytes |
| `./bin/codetok daily` run 3 | 0.53s real | dashboard output, 863 bytes |
| `./bin/codetok daily --json` | 0.49s real | valid JSON, 15 rows, 4124 bytes |
| `./bin/codetok daily --all --json` | 2.29s real | valid JSON, 219 rows, 59793 bytes |
| `./bin/codetok daily --provider claude --json` | 0.05s real | valid JSON, 6 rows, 1664 bytes |
| `./bin/codetok session --json` | 2.28s real | valid JSON, 1550 rows, 1187483 bytes |
| `./bin/codetok session --since 2026-04-15 --until 2026-04-16 --json` | 0.24s real | valid JSON, 114 rows, 85736 bytes |

Default `daily` median wall time is 0.51s. Compared with the recorded local baseline of 5.07s to 5.26s, this is about 90% faster versus either baseline value on this machine and dataset.

## Residual Risk Decision

- EAP-004 is included in this final acceptance pass and remains done after rebase; its workspace evidence records the streaming daily path, and the refreshed benchmark shows the streaming path removes the command-level event-slice allocation.
- EAP-006 is included in this final acceptance pass and remains done after rebase. EAP-004 already removed the materialized daily filter/aggregate path, so EAP-006 required no additional code and no EAP optimization tasks remain open.
- Local timing remains machine and dataset dependent; correctness gates and synthetic metrics are the durable guardrails.
