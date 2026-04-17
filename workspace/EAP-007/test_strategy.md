# EAP-007 Test Strategy

## Focused Checks

Run these before the full repo gates:

```bash
go test ./cmd -run 'Test(CollectUsageEventsFromProviders|RunDaily|RunSession|ResolveDailyDateRange|ResolveSession)' -count=1
go test ./provider/claude ./provider/codex ./provider/kimi ./provider/cursor -run 'Test.*UsageEventsInRange|TestCollectClaudeUsageEventsWithParser_ParsesInParallelAndSorts|TestParseCodexUsageEvents' -count=1
go test ./stats -run 'Test(FilterEventsByDateRange|AggregateEventsByDayWithDimension)' -count=1
go test ./e2e -run TestEventBasedCrossDayAcceptance -count=1
```

These checks cover the range-aware path, exact event timestamp filtering, cross-day sessions, and `--all`/date semantics. The provider package checks also assert `UsageEventCollectMetrics` counts for considered, skipped, parsed, and emitted files/events on synthetic fixtures.

## Benchmarks

Run existing synthetic benchmarks as supporting evidence:

```bash
go test ./provider/claude -run '^$' -bench 'BenchmarkCollectClaudeUsageEventsSynthetic$' -count=1
go test ./provider/codex -run '^$' -bench 'BenchmarkParseCodexUsageEventsSynthetic$' -count=1
```

## Final Gates

Run after all documentation and task-board edits are complete:

```bash
make fmt
make test
make vet
make lint
make build
```

If `golangci-lint` is unavailable, record that `make lint` was skipped because the tool is not installed.

## Built-Binary Smoke

Run only after `make build`:

```bash
./bin/codetok daily
./bin/codetok daily --json
./bin/codetok daily --all --json
./bin/codetok daily --provider claude --json
./bin/codetok session --json
./bin/codetok session --since 2026-04-15 --until 2026-04-16 --json
```

Use `/usr/bin/time -p` for timing evidence and record wall-clock results in `workspace/EAP-007/verification.md` and the plan.
