# Event Aggregation Performance Optimization Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `codetok daily` fast again after the event-based token aggregation change, without losing cross-day event attribution correctness.

**Primary Fix Strategy:** Keep event-level correctness, but stop parsing avoidable historical files and remove avoidable sequential work. First parallelize provider file parsing where it is still sequential, then push the selected date window into provider collection as a candidate-file filter, then only optimize deeper JSON parsing and aggregation if measurements still require it.

**Tech Stack:** Go, Cobra, existing provider registry, provider parsers, `stats` package, local JSONL/CSV files, `os.FileInfo.ModTime`, bounded worker pools.

---

## Writing Principles

This document is a handoff artifact. A future implementer should be able to understand the performance problem, the evidence already gathered, the correctness boundaries, the implementation slices, and the acceptance criteria without reading chat history.

Keep this plan necessary, concrete, and falsifiable. Do not treat manual shell experiments as product behavior. Use code, tests, and repeatable CLI measurements as acceptance evidence.

## Context

The event-based aggregation plan in `docs/plans/2026-04-16-event-based-token-aggregation.md` fixed a correctness bug: `daily` and `session` now attribute token usage by each usage event timestamp instead of by session start time. That correctness must be preserved.

The performance regression appeared after `daily` moved from session-level collection to event-level collection. The current command path loads all historical usage events first, then applies the default 7-day filter:

- `cmd/daily.go`: `runDailyWithProviders` calls `collectUsageEventsFromProviders` before resolving and applying the date window.
- `cmd/collect.go`: `collectUsageEventsFromProviders` eagerly appends all provider events into one `[]provider.UsageEvent`.
- `stats.FilterEventsByDateRange` and `stats.AggregateEventsByDayWithDimension` run after full historical event collection.

The important mental model:

```text
Current shape:
scan all files -> parse all historical events -> filter last 7 days -> aggregate

Target shape:
compute date window -> quickly select candidate files -> parse candidates concurrently -> filter exact event timestamps -> aggregate
```

The selected date window must remain exact at the event level. File metadata is only a coarse candidate filter; it must never replace event timestamp filtering.

## What Was Verified

The investigation artifacts live under the local workspace directory:

`workspace/event-token-perf-2026-04-17/`

The summary report is:

`workspace/event-token-perf-2026-04-17/SUMMARY.md`

Artifact-to-claim map:

- `SUMMARY.md` proves the high-level bottleneck conclusion and lists all supporting artifacts.
- `logs/default.time.log` and `logs/default-second.time.log` prove the basic `./bin/codetok daily` wall-clock baseline.
- `logs/compare-events-*.log` and `logs/compare-sessions-*.log` prove the event path versus session path comparison.
- `profiles/all.cpu.top.txt` and `profiles/all.mem.alloc_space.top.txt` prove the parser-heavy CPU and allocation profile.
- `agents/ccusage-reference.md` records the ccusage comparison.
- `agents/codetok-static-audit.md` records the code-path audit.

Verified behavior on the investigator machine:

| Measurement | Result |
| --- | ---: |
| `./bin/codetok daily` run 1 | 5.26s wall |
| `./bin/codetok daily` run 2 | 5.07s wall |
| Direct profile harness, all providers | 5.93s total |
| Direct profile harness collection time | 5.92s |
| Direct profile harness filter time | 7.3ms |
| Direct profile harness aggregate time | 2.2ms |

Provider-level measurements:

| Provider | Event records | Event collect | Session records | Session collect | Slowdown |
| --- | ---: | ---: | ---: | ---: | ---: |
| all | 88,389 | 5.92s | 2,971 | 2.08s | 2.85x |
| Claude | 24,777 | 3.81s | 1,966 | 0.64s | 5.91x |
| Codex | 62,179 | 1.70s | 896 | 1.03s | 1.65x |
| Kimi | 1,434 | 0.16s | 109 | 0.07s | 2.16x |

Direct profiling showed that the current bottleneck is provider parsing, not dashboard rendering or stats aggregation:

- `encoding/json.Unmarshal` dominated CPU samples.
- `provider/codex.parseCodexUsageEvents` and `provider/claude.parseUsageEvents` dominated application-level parser time.
- Allocation churn is high during parsing, but retained heap after GC is small.

Reference repo notes from a local checkout of `ccusage`:

- `ccusage` preserves event-level attribution for Codex usage.
- Its Claude loader uses streaming line-by-line JSONL processing.
- It does not persist parsed usage event caches between runs.
- Its useful pattern for this task is reducing unnecessary work, not adding a durable usage cache.

Invalid evidence and traps:

- Some early provider-specific timing logs under `workspace/event-token-perf-2026-04-17/logs/{codex,claude,kimi,cursor}.time.log` are invalid because the shell wrapper passed `daily --provider X` as one argument. Use the later `provider-*.time.log` and `compare-*.log` files instead.
- The test-based pprof report in `agents/runtime-profile.md` exercises command tests, not the real local dataset. Treat it as supporting evidence only. The direct profile harness is better evidence for local-data performance.

## Boundary Problems

### Boundary 1: File Metadata Is Only a Candidate Filter

`mtime` can identify files that were probably inactive before the selected window, but it is not the source of truth for token attribution. The source of truth remains each event's timestamp inside the provider log.

Acceptance must include a cross-day session case:

- session starts before `--since`
- file is modified inside the window
- event inside the window is included
- event before the window is excluded

### Boundary 2: Session Creation Time Is Not Portable Enough Alone

The user suggested screening by session creation time and latest modification time. This is directionally correct, but Go's portable `os.FileInfo` exposes modification time, not reliable birth time across all platforms. Provider-specific path dates can also help, especially Codex's `year/month/day` layout.

Implementation should prefer:

- portable `ModTime` for "could this file contain new events in the selected window?"
- provider path dates where they are part of the documented local storage layout
- exact event timestamp filtering after parsing candidate files

### Boundary 3: Daily and Session Share Event Semantics

`daily` and `session` both depend on usage events after the event-based change. A collection optimization for `daily` must not silently change `session --since/--until` semantics.

Range-aware collection should be introduced as shared collection infrastructure, but command adoption must be explicit:

- `daily` should adopt range-aware collection first because its default 7-day window is the measured slow path.
- `session` should adopt the same range-aware collection for `--since` and `--until` in the same change only when parity tests prove that sessions with in-window events are preserved.
- `session` without a date range may keep full collection behavior.

### Boundary 4: Reporting Commands Must Stay Local-Only

This optimization must not add remote provider API calls. `daily`, `session`, and `cursor activity` remain local-file readers. Cursor network commands remain limited to explicit Cursor login/status/sync behavior.

## Follow-up Work

### Task 1: Add Reusable Performance Baseline and Correctness Fixtures

Create repeatable measurements before changing parser behavior.

Implementation notes:

- Keep one-off large local profiles in `workspace/`; do not commit machine-specific profile artifacts.
- Add a stable benchmark or test harness that builds synthetic local provider data in a temp directory, so performance assertions do not depend only on one developer machine.
- Add small fixture tests that prove range filtering preserves event timestamp correctness.
- Add tests for cross-day sessions where file/session metadata and event timestamps disagree.
- Prefer package tests for deterministic behavior and manual CLI timing for performance acceptance.

Suggested tests:

- `cmd/daily_test.go`: default 7-day window still filters by event timestamp, not session/file date.
- `cmd/session_test.go`: sessions with in-window events remain visible even when session start is old.
- Provider package tests: candidate filtering includes files modified in-window and excludes files safely outside the window.

Task acceptance:

- Existing correctness tests still pass before optimization.
- New tests fail if candidate filtering uses file/session time as the final attribution source.
- A repeatable fixture or benchmark can show how many files were considered, parsed, skipped by metadata, and finally included by event timestamp.
- Manual baseline is recorded in a new workspace run before and after optimization as supporting evidence, not as the only pass/fail gate.

### Task 2: Parallelize Claude Usage Event Parsing

Claude event collection is currently sequential:

```go
for _, path := range paths {
    parsed, err := parseUsageEvents(path, pathToSlug[path])
    ...
}
```

Change `provider/claude.(*Provider).CollectUsageEvents` to use bounded parallel parsing, matching the existing style used by session collection and Codex event collection.

Implementation notes:

- Reuse `provider.ParseUsageEventsParallel` or add a typed helper if needed.
- Preserve local-only behavior and missing-file tolerance.
- Do not rely on input order; downstream aggregators must sort or group deterministically.
- Confirm `session` output remains deterministic because session aggregation sorts its output.

Task acceptance:

- `go test ./provider/claude`
- `go test ./cmd -run 'TestRunDaily|TestRunSession'`
- Manual `./bin/codetok daily --provider claude` improves materially against the pre-change baseline on the same dataset.
- Results for `./bin/codetok daily --provider claude --json` match pre-change totals on the same data.

### Task 3: Add Range-Aware Candidate File Filtering

Move the selected date window closer to provider collection so default `daily` does not parse full history.

Proposed shape:

```go
type UsageEventCollectOptions struct {
    Since time.Time
    Until time.Time
    Location *time.Location
}

type RangeAwareUsageEventProvider interface {
    Provider
    CollectUsageEventsInRange(baseDir string, opts UsageEventCollectOptions) ([]UsageEvent, error)
}
```

Alternative names are acceptable, but the architecture should make the range explicit and keep legacy `CollectUsageEvents` available during migration.

Candidate filtering rules:

- If no range is requested, preserve current full-history behavior.
- If `--all` is set, preserve current full-history behavior.
- If a file's `ModTime` is before the start of the selected date window, it may be skipped only when provider semantics make that safe.
- If provider path layout encodes a date, use it only as a candidate shortcut, not as final attribution.
- Always parse candidate files and filter exact usage events by `UsageEvent.Timestamp`.

Provider-specific guidance:

- Claude: use JSONL file `ModTime` as the main candidate filter because paths are project/session based.
- Codex: use the `year/month/day` path layout plus `ModTime`; keep files that could contain cross-day continued usage.
- Kimi: use `wire.jsonl` `ModTime` and keep exact event filtering.
- Cursor: CSV rows are already row-oriented; range filtering can happen while reading rows, but explicit `--cursor-dir` must stay authoritative.

Task acceptance:

- Default `codetok daily` no longer parses files that are provably inactive before the default 7-day window.
- Cross-day sessions remain correct: old session files with in-window events are still included.
- `codetok session --since/--until` uses the range-aware path only after tests prove it preserves sessions with in-window events from old session files.
- `codetok session` without a date range may keep full-history collection unless a later task proves a safe optimization.
- `--all` returns the same JSON totals as the pre-optimization full-history event path.
- `--since`, `--until`, `--days`, and `--timezone` keep existing mutual exclusion and date semantics.
- Manual timing for `./bin/codetok daily` improves materially against the recorded baseline.

### Task 4: Add Streaming or Direct-to-Aggregator Daily Collection If Needed

If Tasks 2 and 3 do not meet performance acceptance, avoid materializing all selected events before aggregation.

Possible shape:

```go
type UsageEventConsumer func(provider.UsageEvent) error
```

Provider collection can call the consumer for each candidate event. `daily` can then filter and aggregate in one pass. This is a larger architecture change and should not be the first implementation step unless measurements prove it is needed.

Task acceptance:

- `daily` does not require building one full `[]UsageEvent` for all selected providers.
- JSON and dashboard output remain identical to the materialized event path.
- Errors from provider parsing still include provider context.
- Memory allocation decreases in pprof `alloc_space` compared with the range-aware materialized path.

### Task 5: Reduce Codex JSON Parsing Churn If Needed

Codex is already parallel, but CPU and allocations are high. Optimize only after range filtering, so the parser is optimized for the remaining real workload.

Implementation notes:

- Avoid generic map-based model extraction on every `event_msg` when typed fields are enough.
- Decode only event-type-specific fields.
- Keep `last_token_usage` and cumulative `total_token_usage` delta behavior unchanged.
- Keep model fallback behavior covered by tests.

Task acceptance:

- `go test ./provider/codex`
- Existing Codex delta tests still pass:
  - last-token usage emits one event
  - cumulative totals produce deltas across days
  - resets do not produce negative deltas
  - model fallback behavior remains stable
- CPU and allocation profiles show lower `encoding/json` cumulative cost for Codex on the same data.

### Task 6: Collapse Date Filtering and Aggregation Passes Only After Parser Work

This is a cleanup optimization, not the primary fix. Current evidence shows filtering plus aggregation costs under 10ms on the investigated dataset.

Implementation notes:

- Combine date range checks and daily bucket aggregation only if the event slice remains materialized.
- Preserve distinct session counting by date/group.
- Preserve `GroupBy`, `Group`, `ProviderName`, and `Providers` JSON semantics.

Task acceptance:

- `go test ./stats`
- `go test ./cmd -run TestRunDaily`
- No user-visible JSON field changes.
- Any performance gain here is treated as incremental, not as proof that provider parsing was fixed.

## Acceptance Standard

This work is complete only when all of the following are true.

Correctness:

- `codetok daily` still attributes token usage by event timestamp in the selected timezone.
- `codetok session --since/--until` still includes sessions with in-window usage events even if the session started before the window.
- Cross-day sessions are tested and pass for Codex, Claude, and Kimi where fixtures exist.
- JSON totals for `--all` match the pre-optimization event path on the same fixture data.
- Reporting commands remain local-only and do not trigger Cursor login/sync or remote provider APIs.

Performance:

- A stable fixture or benchmark demonstrates that the default-window path parses fewer files than the full-history path while producing the same in-window results as exact event timestamp filtering.
- A benchmark or instrumentation report records considered files, metadata-skipped files, parsed files, emitted events, filtered events, and final rows.
- On the same local dataset used for baseline, `./bin/codetok daily` should improve materially; target at least 50% wall-clock improvement as a local sanity check, not as the only portable gate.
- `./bin/codetok daily --provider claude` improves materially after parallel parsing.
- If the stable fixture or local-data evidence shows parser collection still dominates after Tasks 2 and 3, continue to Task 4 or document why the remaining work is unavoidable.

Verification commands:

```bash
make fmt
make test
make vet
make lint   # if golangci-lint is installed
make build
./bin/codetok daily
./bin/codetok daily --json
./bin/codetok daily --all --json
./bin/codetok session --json
```

Manual CLI checks must run after `make build`; do not use `go run . ...` as acceptance evidence for the built binary.

## Final Acceptance Evidence

EAP-007 final acceptance was run in `.worktrees/eap-007-final-acceptance` on 2026-04-17 after rebasing onto `origin/main` with EAP-004 and EAP-006. The detailed evidence is recorded in `workspace/EAP-007/verification.md`.

Fresh verification completed:

- focused command, provider, stats, and cross-day e2e checks passed
- synthetic Claude, Codex, and materialized-vs-streaming daily benchmarks passed
- provider metric assertions cover considered, skipped, parsed, and emitted counts on synthetic fixtures
- `go clean -testcache` followed by `make test` passed without cached package results
- `make fmt`, `make vet`, `make lint`, and `make build` passed
- built-binary smoke checks passed for `daily`, `daily --json`, `daily --all --json`, `daily --provider claude --json`, `session --json`, and `session --since 2026-04-15 --until 2026-04-16 --json`

Current local timing on the same local data family:

| Command | Result |
| --- | ---: |
| `./bin/codetok daily` run 1 | 0.48s real |
| `./bin/codetok daily` run 2 | 0.51s real |
| `./bin/codetok daily` run 3 | 0.53s real |
| `./bin/codetok daily --json` | 0.49s real |
| `./bin/codetok daily --all --json` | 2.29s real |
| `./bin/codetok daily --provider claude --json` | 0.05s real |
| `./bin/codetok session --json` | 2.28s real |
| `./bin/codetok session --since 2026-04-15 --until 2026-04-16 --json` | 0.24s real |

The default `daily` median is 0.51s, compared with the recorded 5.07s/5.26s baseline from the original investigation. That is about 90% faster on this machine and dataset.

Conditional task decision:

- EAP-004 is included in this final acceptance pass and remains done after rebase; its workspace evidence records the streaming daily path, and the refreshed materialized-vs-streaming benchmark confirms the command-level event-slice allocation reduction.
- EAP-006 is included in this final acceptance pass and remains done after rebase. EAP-004 already removed the materialized daily filter/aggregate pass, so EAP-006 required no additional code and no EAP optimization tasks remain open.

## Problems Encountered

- The first invalid provider timing attempt passed `daily --provider codex` as one shell argument. Future timing scripts must pass command arguments explicitly.
- Test-based pprof can miss the real bottleneck if tests use tiny fake providers. Prefer a direct local-data harness or a real CLI pprof hook for performance diagnosis.
- It is tempting to use file `ModTime` as if it were event time. Do not do that. Metadata can exclude obvious non-candidates only; event timestamp remains authoritative.
- `--all` is an important guardrail. Range-aware optimizations must be bypassed or made equivalent for full-history requests.
- Provider order and parser concurrency can change event ordering. Commands must sort or aggregate deterministically instead of relying on parse order.

## Source Artifacts

- Investigation summary: `workspace/event-token-perf-2026-04-17/SUMMARY.md`
- Static audit: `workspace/event-token-perf-2026-04-17/agents/codetok-static-audit.md`
- ccusage reference: `workspace/event-token-perf-2026-04-17/agents/ccusage-reference.md`
- Runtime profile report: `workspace/event-token-perf-2026-04-17/agents/runtime-profile.md`
- Existing correctness plan: `docs/plans/2026-04-16-event-based-token-aggregation.md`
