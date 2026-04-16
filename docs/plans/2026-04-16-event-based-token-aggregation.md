# Event-Based Token Aggregation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix cross-day token attribution so `daily` and `session` reports aggregate usage by each token event's timestamp instead of the session start date.

**Architecture:** Introduce a provider-level `UsageEvent` model and make providers translate local logs into timestamped token deltas. `daily` groups events by calendar date in a selected timezone; `session` groups the same filtered events by session ID, preserving session views without using session start time as the filtering boundary.

**Tech Stack:** Go, Cobra, existing provider registry, existing `stats` package, local JSONL/CSV parsers, `time.LoadLocation`

---

## Context

Current `codetok` aggregates through `provider.SessionInfo`. `daily` filters and groups by `SessionInfo.StartTime`, so a session created yesterday but used today still reports today's token usage under yesterday. If the session started outside the selected window, today's continued usage can be filtered out entirely.

This is a data-model problem, not only a Codex parser bug. Codex, Claude Code, and Kimi all currently collapse many timestamped records into one `SessionInfo` before `daily` sees them.

`ccusage/codex` uses a better model:

- It parses each Codex `token_count` into a timestamped event.
- It prefers `last_token_usage` when available.
- If only cumulative `total_token_usage` exists, it subtracts the previous cumulative total to recover the event delta.
- It groups daily output by `event.timestamp` in a selected timezone.
- It reads `CODEX_HOME` before falling back to `~/.codex`.

This plan applies that event-first design to `codetok` while keeping the existing CLI shape.

## Target Behavior

- `codetok daily` attributes token usage to the local calendar day on which the token event occurred.
- `codetok session --since/--until` includes sessions that have usage events in the selected date range, even if the session started earlier.
- `DailyStats.Sessions` means the number of distinct session IDs that contributed usage to that date/group, not the number of token events.
- `--timezone IANA/Name` controls date grouping and date filters.
- The default timezone is the user's local timezone.
- Codex default source resolution uses `$CODEX_HOME/sessions` when `CODEX_HOME` is set, otherwise `~/.codex/sessions`.
- Existing JSON field names remain stable unless a later schema migration explicitly changes them.

## Non-Goals

- Do not add remote provider API calls.
- Do not add cost calculation.
- Do not change Cursor CSV semantics beyond making Cursor compatible with the new event collector.
- Do not remove the existing `provider.SessionInfo` type until all commands no longer need it.
- Do not change command names or flag mutual-exclusion rules except adding `--timezone`.

## Proposed Data Model

Add a timestamped usage event in `provider/provider.go`:

```go
type UsageEvent struct {
	ProviderName string
	ModelName    string
	SessionID    string
	Title        string
	WorkDirHash  string
	Timestamp    time.Time
	TokenUsage   TokenUsage
	SourcePath   string
	EventID      string
}
```

Add an optional provider interface:

```go
type UsageEventProvider interface {
	Provider
	CollectUsageEvents(baseDir string) ([]UsageEvent, error)
}
```

Keep `Provider.CollectSessions` during migration. Command collection can use `CollectUsageEvents` when available and fall back to converting sessions to one event only for providers not yet migrated. The acceptance target for this change is that Codex, Claude, Kimi, and Cursor all provide native events before `daily` switches to event aggregation.

## Dependency Graph

1. Task 1 is the foundation and must land first.
2. Tasks 2, 3, and 4 depend on Task 1 and can be implemented in parallel.
3. Task 5 depends on Tasks 2, 3, and 4.

```text
Task 1: Core event model + timezone stats
  -> Task 2: Codex events
  -> Task 3: Claude events
  -> Task 4: Kimi events
Tasks 2 + 3 + 4
  -> Task 5: Command integration + docs + acceptance
```

## Validation Rules

Every task must follow this loop:

1. Write the failing test first.
2. Run the narrow test and confirm it fails for the expected reason.
3. Implement the minimum code needed.
4. Run the narrow test and confirm it passes.
5. Run the package-level test for touched packages.
6. If any test fails unexpectedly, fix within the same task before moving on.

Final acceptance requires:

- `make fmt`
- `make test`
- `make vet`
- `make lint` when `golangci-lint` is installed
- `make build`
- manual CLI checks with `./bin/codetok`

## Task 1: Core UsageEvent Model, Timezone Date Utilities, and Event Aggregation

**Files:**

- Modify: `provider/provider.go`
- Modify: `cmd/collect.go`
- Modify: `cmd/daily.go`
- Create: `stats/events.go`
- Create: `stats/events_test.go`
- Modify: `stats/aggregator_test.go` only if shared helpers move
- Modify: `cmd/daily_test.go`

**Step 1: Write the failing stats tests**

Add tests proving event-level daily attribution and timezone grouping.

```go
func TestAggregateEventsByDayWithDimension_SplitsSameSessionAcrossDays(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	events := []provider.UsageEvent{
		{
			ProviderName: "codex",
			SessionID:    "same-session",
			ModelName:    "gpt-5.4",
			Timestamp:    time.Date(2026, 4, 15, 23, 50, 0, 0, loc),
			TokenUsage:   provider.TokenUsage{InputOther: 100, Output: 10},
		},
		{
			ProviderName: "codex",
			SessionID:    "same-session",
			ModelName:    "gpt-5.4",
			Timestamp:    time.Date(2026, 4, 16, 0, 10, 0, 0, loc),
			TokenUsage:   provider.TokenUsage{InputOther: 200, Output: 20},
		},
	}

	got := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, loc)

	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %#v", len(got), got)
	}
	if got[0].Date != "2026-04-15" || got[0].TokenUsage.Total() != 110 {
		t.Fatalf("first row mismatch: %#v", got[0])
	}
	if got[1].Date != "2026-04-16" || got[1].TokenUsage.Total() != 220 {
		t.Fatalf("second row mismatch: %#v", got[1])
	}
}
```

Add a timezone boundary test:

```go
func TestAggregateEventsByDayWithDimension_UsesRequestedTimezone(t *testing.T) {
	utc := time.UTC
	shanghai := time.FixedZone("UTC+8", 8*3600)
	events := []provider.UsageEvent{{
		ProviderName: "codex",
		SessionID:    "s1",
		Timestamp:    time.Date(2026, 4, 15, 18, 0, 0, 0, utc),
		TokenUsage:   provider.TokenUsage{InputOther: 1},
	}}

	gotUTC := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, utc)
	gotShanghai := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, shanghai)

	if gotUTC[0].Date != "2026-04-15" {
		t.Fatalf("UTC date = %q, want 2026-04-15", gotUTC[0].Date)
	}
	if gotShanghai[0].Date != "2026-04-16" {
		t.Fatalf("Shanghai date = %q, want 2026-04-16", gotShanghai[0].Date)
	}
}
```

**Step 2: Run tests to verify failure**

Run:

```bash
go test ./stats -run 'TestAggregateEvents'
```

Expected: FAIL because `UsageEvent` and event aggregators do not exist.

**Step 3: Implement core event types and aggregation**

Add `UsageEvent` and `UsageEventProvider` in `provider/provider.go`.

Create `stats/events.go` with:

- `AggregateEventsByDayWithDimension(events []provider.UsageEvent, dimension AggregateDimension, loc *time.Location) []provider.DailyStats`
- `FilterEventsByDateRange(events []provider.UsageEvent, sinceDate, untilDate string, loc *time.Location) []provider.UsageEvent`
- `EventSessionCount` behavior via distinct `SessionID` per daily group
- `eventGroupNameForDimension` using the existing CLI/model semantics

Implementation rules:

- Normalize nil `loc` to `time.Local`.
- Date key is `event.Timestamp.In(loc).Format("2006-01-02")`.
- `DailyStats.Sessions` increments once per distinct session/container key per date/group.
- Build that key from `SessionID` when present; otherwise use the session-level `SourcePath`.
- Do not include per-event identifiers such as `EventID` in the session-count fallback, because that would inflate `Sessions` to event count for logs missing `SessionID`.
- Preserve `ProviderName`, `GroupBy`, `Group`, and multi-provider behavior from session aggregation.

**Step 4: Add collector helper tests**

In `cmd/collect_test.go`, add a fake provider implementing `UsageEventProvider` and prove `collectUsageEventsFromProviders` uses native events.

Add a fallback provider test only to keep Cursor or future providers compiling during migration:

```go
func TestCollectUsageEventsFromProviders_UsesNativeEvents(t *testing.T) {
	// provider returns one event; assert it is collected unchanged.
}

func TestCollectUsageEventsFromProviders_FallsBackToSessionEvent(t *testing.T) {
	// provider only has CollectSessions; assert one event is synthesized from StartTime.
}
```

**Step 5: Add timezone flag parsing tests**

In `cmd/daily_test.go`, add tests for:

- empty `--timezone` resolves to `time.Local`
- `--timezone Asia/Shanghai` resolves successfully
- invalid timezone returns a concise error
- default `--days` range uses local day boundaries, not UTC day boundaries

**Step 6: Run tests**

Run:

```bash
go test ./provider ./stats ./cmd -run 'Test(CollectUsageEvents|AggregateEvents|ResolveDaily|ResolveTimezone)'
```

Expected: PASS.

**Step 7: Commit**

```bash
git add provider/provider.go stats/events.go stats/events_test.go cmd/collect.go cmd/collect_test.go cmd/daily.go cmd/daily_test.go
git commit -m "feat: add event-based aggregation foundation"
```

## Task 2: Codex Usage Events, Cumulative Delta Handling, Model Context, and CODEX_HOME

**Files:**

- Modify: `provider/codex/parser.go`
- Modify: `provider/codex/parser_test.go`
- Modify: `cmd/collect_test.go` only if Codex source resolution is tested at command level

**Step 1: Write failing Codex event tests**

Add tests that prove:

- `last_token_usage` creates one event without subtracting.
- `total_token_usage` creates deltas by subtracting the previous total.
- Two token counts across two dates produce two events with the original event timestamps.
- `turn_context.payload.model` is used when `token_count` lacks model metadata.
- `CODEX_HOME` is honored when no `--codex-dir` is passed.

Example test skeleton:

```go
func TestParseCodexUsageEvents_TotalUsageDeltasAcrossDays(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-test.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-04-15T23:50:00Z","type":"turn_context","payload":{"model":"gpt-5.4"}}`,
		`{"timestamp":"2026-04-15T23:55:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}}}}`,
		`{"timestamp":"2026-04-16T00:10:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":250,"output_tokens":450,"reasoning_output_tokens":0,"total_tokens":1950}}}}`,
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	events, err := parseCodexUsageEvents(path)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, 800, events[0].TokenUsage.InputOther)
	require.Equal(t, 200, events[0].TokenUsage.InputCacheRead)
	require.Equal(t, 300, events[0].TokenUsage.Output)
	require.Equal(t, 450, events[1].TokenUsage.InputOther)
	require.Equal(t, 50, events[1].TokenUsage.InputCacheRead)
	require.Equal(t, 150, events[1].TokenUsage.Output)
}
```

Use the project test style instead of `require` if this repo avoids external test helpers.

**Step 2: Run tests to verify failure**

Run:

```bash
go test ./provider/codex -run 'Test(ParseCodexUsageEvents|CollectCodexUsageEvents|CodexHome)'
```

Expected: FAIL because event parsing and `CODEX_HOME` support do not exist.

**Step 3: Implement Codex event parsing**

Add:

- `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)`
- `func parseCodexUsageEvents(path string) ([]provider.UsageEvent, error)`

Parsing rules:

- Scan the same session file set as `CollectSessions`.
- Keep `previousTotals` per file.
- Prefer `info.last_token_usage` if present.
- If `last_token_usage` is missing and `total_token_usage` exists, subtract `previousTotals`.
- Update `previousTotals` whenever `total_token_usage` exists.
- Skip zero deltas.
- Treat `cached_input_tokens` as part of input: `InputOther = input_tokens - cached_input_tokens`.
- Do not add `reasoning_output_tokens` to `Output`; Codex output already includes it.
- Track `currentModel` from `turn_context`, `event_msg.payload.model`, `info.model`, and existing raw JSON model paths.
- Use source-relative path or `session_meta.id` as `SessionID`; prefer a stable value that does not change when a file contains multiple `session_meta` records.
- Use event timestamp from the `token_count` JSONL line.

Add source resolution:

```text
if --codex-dir/baseDir is set:
    use it exactly
else if CODEX_HOME is set:
    use $CODEX_HOME/sessions
else:
    use ~/.codex/sessions
```

**Step 4: Preserve existing session parser tests**

Run:

```bash
go test ./provider/codex
```

Expected: PASS. Existing `parseCodexSession` behavior can remain for `session` until Task 5.

**Step 5: Commit**

```bash
git add provider/codex/parser.go provider/codex/parser_test.go
git commit -m "feat: collect codex usage events"
```

## Task 3: Claude Code Usage Events With Streaming Deduplication

**Files:**

- Modify: `provider/claude/parser.go`
- Modify: `provider/claude/parser_test.go`

**Step 1: Write failing Claude event tests**

Add tests proving:

- Each assistant message with `message.usage` becomes one `UsageEvent`.
- Events use the assistant message timestamp, not the first user timestamp.
- Duplicate streaming records with the same `message.id + requestId` keep the latest usage.
- Cross-day assistant messages from one session produce events on both days.
- Subagent files are included with native event collection just like session collection.

Example expected shape:

```go
func TestParseClaudeUsageEvents_CrossDayAssistantMessages(t *testing.T) {
	// Build one JSONL with user on 2026-04-15 and assistant usage on
	// both 2026-04-15 and 2026-04-16.
	// Assert two UsageEvents with different timestamps and the same SessionID.
}
```

**Step 2: Run tests to verify failure**

Run:

```bash
go test ./provider/claude -run 'Test(ParseClaudeUsageEvents|CollectClaudeUsageEvents)'
```

Expected: FAIL because Claude event collection does not exist.

**Step 3: Implement Claude event parsing**

Add:

- `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)`
- `func parseUsageEvents(path, projectSlug string) ([]provider.UsageEvent, error)`

Parsing rules:

- Use existing `collectPaths` so default and explicit Claude directories behave the same.
- Only assistant events with non-nil `message.usage` create usage events.
- Deduplicate by `message.id + requestId` using the latest usage for that key.
- If no dedup key exists, treat each assistant usage entry as unique.
- Preserve `ProviderName`, `SessionID`, `ModelName`, `WorkDirHash`, `Title` when available.
- Event timestamp is the assistant event timestamp.

Implementation detail:

- To keep the latest usage per key while preserving timestamp, store a struct containing `provider.UsageEvent` plus the latest parsed timestamp.
- Sum nothing in the parser; each deduped assistant usage entry is its own event.

**Step 4: Run tests**

Run:

```bash
go test ./provider/claude
```

Expected: PASS.

**Step 5: Commit**

```bash
git add provider/claude/parser.go provider/claude/parser_test.go
git commit -m "feat: collect claude usage events"
```

## Task 4: Kimi Usage Events From StatusUpdate Records

**Files:**

- Modify: `provider/kimi/parser.go`
- Modify: `provider/kimi/parser_test.go`

**Step 1: Write failing Kimi event tests**

Add tests proving:

- Each `StatusUpdate` with `token_usage` becomes one `UsageEvent`.
- Event timestamp is the `StatusUpdate` line timestamp.
- Cross-day `StatusUpdate` records from one session split correctly at aggregation time.
- `metadata.json` model fields and log fallback model lookup still populate model names.
- Existing `parseSession` tests still pass.

Example expected shape:

```go
func TestParseKimiUsageEvents_StatusUpdatesUseOwnTimestamps(t *testing.T) {
	// Build one wire.jsonl with TurnBegin on 2026-04-15,
	// one StatusUpdate on 2026-04-15, and another StatusUpdate on 2026-04-16.
	// Assert two UsageEvents with the StatusUpdate timestamps.
}
```

**Step 2: Run tests to verify failure**

Run:

```bash
go test ./provider/kimi -run 'Test(ParseKimiUsageEvents|CollectKimiUsageEvents)'
```

Expected: FAIL because Kimi event collection does not exist.

**Step 3: Implement Kimi event parsing**

Add:

- `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)`
- `func parseUsageEvents(sessionPath, workDirHash string, sessionModelIndex map[string]string) ([]provider.UsageEvent, error)`

Parsing rules:

- Reuse existing session discovery logic.
- Read `metadata.json` once for `SessionID`, `Title`, and model fields.
- For every valid `StatusUpdate`, emit one event using that line's timestamp.
- Treat `StatusUpdate.token_usage` as an already-incremental usage record, matching current tests.
- If future real logs prove Kimi reports cumulative counters, add a separate delta-detection task instead of guessing in this change.

**Step 4: Run tests**

Run:

```bash
go test ./provider/kimi
```

Expected: PASS.

**Step 5: Commit**

```bash
git add provider/kimi/parser.go provider/kimi/parser_test.go
git commit -m "feat: collect kimi usage events"
```

## Task 5: Switch Daily and Session Commands to Event Aggregation, Update Docs, and Add Acceptance Tests

**Files:**

- Modify: `cmd/daily.go`
- Modify: `cmd/session.go`
- Modify: `cmd/collect.go`
- Modify: `cmd/daily_test.go`
- Create or modify: `cmd/session_test.go`
- Modify: `e2e/e2e_test.go`
- Modify: `README.md`
- Modify: `README_zh.md`
- Modify: `docs/design.md` only if project architecture docs need updating

**Step 1: Write failing command tests**

Add command-level tests proving:

- `daily` splits a same-session cross-day provider event into the correct days.
- `daily --timezone Asia/Shanghai` uses Shanghai date boundaries.
- default `daily --days 1` uses the local date key, not UTC.
- `session --since 2026-04-16` includes a session that started on `2026-04-15` but has a usage event on `2026-04-16`.
- `session` output totals are based only on events that passed the date filter.

Expected session semantics:

- `Date` in session JSON/table is the first included event date in the selected timezone.
- `EndTime` or last activity, if exposed internally, is the latest included event timestamp.
- `Turns` can remain best-effort from legacy session metadata until a future event-level turn model exists.

**Step 2: Run tests to verify failure**

Run:

```bash
go test ./cmd -run 'Test(Daily|Session).*Event'
```

Expected: FAIL because commands still aggregate `SessionInfo`.

**Step 3: Switch `daily` to events**

In `runDaily`:

1. Resolve timezone before filtering.
2. Collect usage events via `collectUsageEvents(cmd)`.
3. Resolve date window as date keys in the selected timezone.
4. Filter events by event date.
5. Aggregate with `stats.AggregateEventsByDayWithDimension`.

Keep existing flag constraints:

- `--all` conflicts with `--days`, `--since`, `--until`.
- `--days` conflicts with `--since`, `--until`.

Add:

```bash
--timezone Asia/Shanghai
```

Default: `time.Local`.

**Step 4: Switch `session` to events**

Add a session event aggregator in `cmd/session.go` or `stats/events.go`:

- Group by provider + session ID.
- Sum event token usage.
- Track first and last included event timestamps.
- Preserve model and title from events when available.
- Filter by event date in timezone before grouping.

Do not include a session in a date-filtered report just because it started in range; include it only when at least one usage event is in range.

**Step 5: Make Cursor compatible**

Cursor CSV rows already represent timestamped usage records. Implement `CollectUsageEvents` for Cursor by mapping each parsed CSV row to a `UsageEvent`, or by sharing the CSV parsing result with both session and event collectors.

Run:

```bash
go test ./provider/cursor
```

Expected: PASS.

**Step 6: Add e2e cross-day fixtures**

Add or generate temporary fixtures for:

- Codex cumulative `token_count` crossing midnight.
- Claude assistant usage crossing midnight.
- Kimi `StatusUpdate` crossing midnight.

Add e2e tests that invoke:

```bash
codetok daily --json --since 2026-04-15 --until 2026-04-16 --timezone UTC
codetok session --json --since 2026-04-16 --until 2026-04-16 --timezone UTC
```

Expected:

- daily has separate rows for `2026-04-15` and `2026-04-16`
- `2026-04-16` includes only the usage events that occurred on that date
- session report includes the cross-day session when filtering for `2026-04-16`

**Step 7: Update docs**

Update README and README_zh to state:

- reports aggregate local log usage events
- `daily` uses event timestamps, not session creation time
- `session` filters by event date and groups matching events by session
- default timezone is local
- `--timezone` accepts IANA names
- Codex default source honors `$CODEX_HOME/sessions`

**Step 8: Run targeted tests**

Run:

```bash
go test ./provider/codex ./provider/claude ./provider/kimi ./provider/cursor ./stats ./cmd
go test ./e2e -run 'Test.*(Daily|Session|CrossDay|Codex|Claude|Kimi)'
```

Expected: PASS.

**Step 9: Run final validation**

Run:

```bash
make fmt
make test
make vet
make lint
make build
```

Expected:

- `make fmt`, `make test`, `make vet`, and `make build` pass.
- `make lint` passes when `golangci-lint` is installed; if not installed, record the skipped state from the Makefile behavior.

Manual checks:

```bash
./bin/codetok daily --provider codex --json --timezone Asia/Shanghai
./bin/codetok daily --provider codex --since 2026-04-16 --until 2026-04-16 --timezone Asia/Shanghai --unit raw
./bin/codetok session --provider codex --since 2026-04-16 --until 2026-04-16 --timezone Asia/Shanghai --json
CODEX_HOME=/tmp/nonexistent ./bin/codetok daily --provider codex --json
```

Expected:

- same-session cross-day usage appears on the actual event day
- date filters apply to event dates
- invalid or missing Codex roots remain local-only and do not call remote APIs
- command help documents the timezone and event-based semantics

**Step 10: Commit**

```bash
git add cmd stats provider README.md README_zh.md e2e/e2e_test.go docs/design.md
git commit -m "feat: aggregate usage by token event timestamps"
```

## Risk Register

| Risk | Impact | Mitigation |
|---|---|---|
| Codex `session_meta.id` can change within a file | Session grouping may look unstable | Use relative rollout file path as stable session ID unless a single consistent metadata ID exists. |
| Kimi `StatusUpdate.token_usage` may be cumulative in some versions | Kimi could overcount if real logs differ from current tests | Keep current incremental assumption for this change; add a focused fixture if real logs prove cumulative behavior. |
| `session` output date semantics change | Users may see different session dates | Document that session date now reflects first matching usage event in the selected range. |
| Timezone default change from UTC to local affects historical daily rows | Daily reports shift for users outside UTC | Treat as correctness fix; expose `--timezone UTC` for old UTC-style reports. |
| Event model increases memory usage | Very large logs may allocate more events than session summaries | Keep parser streaming where practical; only store event records needed for aggregation. |

## Acceptance Checklist

- [ ] Cross-day Codex cumulative usage is split by token event timestamp.
- [ ] Cross-day Claude assistant usage is split by assistant message timestamp.
- [ ] Cross-day Kimi `StatusUpdate` usage is split by status update timestamp.
- [ ] `daily --timezone Asia/Shanghai` groups by Shanghai calendar dates.
- [ ] `daily --timezone UTC` groups by UTC calendar dates.
- [ ] `session --since/--until` filters by usage event date, not session start date.
- [ ] `DailyStats.Sessions` counts distinct sessions per day/group.
- [ ] `CODEX_HOME` is honored when `--codex-dir` is not set.
- [ ] README and README_zh explain event-based aggregation and timezone behavior.
- [ ] `make fmt`, `make test`, `make vet`, and `make build` pass.
