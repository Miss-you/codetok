# Event-Based Token Aggregation Task Board

## Source Design

- Source: `docs/plans/2026-04-16-event-based-token-aggregation.md`
- Initialized: 2026-04-16 15:36 CST
- Approval signal: source design is committed in `main` and this task board was requested for implementation breakdown.
- Scope: split the approved event-based token aggregation plan into claimable tasks for `provider/`, `stats/`, `cmd/`, `e2e/`, README docs, and validation gates.
- Non-goals: do not implement any task from this board while creating it; do not add remote provider API calls; do not change command names or existing flag mutual-exclusion rules except the planned `--timezone` addition.

## Status Legend

- `todo`: not claimed and not started.
- `claimed`: owner has claimed the task, recorded a workspace, and is preparing work.
- `research`: owner is gathering local evidence before implementation.
- `spec`: owner is writing or refining tests and acceptance details.
- `implementing`: owner is changing code or docs.
- `verifying`: owner is running validation and fixing failures.
- `review`: task is ready for review or awaiting review feedback.
- `blocked`: task cannot proceed; `Notes` must include `resume_to=<state>` and the blocker.
- `done`: implementation and listed `Done When` checks are complete with evidence.

## Dependency Rules

- Only claim tasks in `todo` when every hard dependency in `Depends On` is `done`.
- `EBTA-001` is the foundation for event types and stats aggregation.
- `EBTA-002` can proceed after `EBTA-001`; provider event tasks can proceed in parallel after `EBTA-001`.
- `EBTA-007` and `EBTA-008` must wait until the collector bridge and all native provider event tasks are done, including Cursor compatibility.
- `EBTA-009` validates cross-command behavior after daily and session integration are both done.
- `EBTA-010` is the final documentation and release gate task after implementation and e2e behavior are complete.
- If the source design changes, refresh this board before claiming additional tasks.

## Task Table

| ID | Title | Goal | Depends On | Parallel | Status | Owner | Claimed At | Workspace | Change | Done When | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| EBTA-001 | Core event model and stats aggregation | Add `provider.UsageEvent`, `provider.UsageEventProvider`, timezone-aware event filtering, and daily aggregation that counts distinct sessions per date/group. | none | No, foundation | done | Codex | 2026-04-16 15:53 CST | `workspace/EBTA-001/` | `event-based-token-aggregation-core` | Event aggregation tests cover cross-day split, timezone grouping, model/CLI grouping, and distinct session counts; `go test ./stats -run TestAggregateEvents` passes. | Done. Verification passed: `go test ./stats -run TestAggregateEvents`; `go test ./provider ./stats`; `make fmt`; `make test`; `make vet`; `make build`; `make lint`. Review notes: `workspace/EBTA-001/review.md`. |
| EBTA-002 | Collector bridge and timezone helpers | Add command collection helpers that prefer native `CollectUsageEvents` and synthesize fallback events for legacy providers; add reusable timezone and date-window resolution tests. | EBTA-001 | Yes, after foundation | todo | unclaimed | - | - | new | `collectUsageEventsFromProviders` uses native events unchanged, falls back to session events when needed, and timezone resolution rejects invalid IANA names; targeted `cmd` tests pass. | First gate: `go test ./cmd -run TestCollectUsageEvents`. Touches `cmd/collect.go`, `cmd/collect_test.go`, `cmd/daily.go`, `cmd/daily_test.go`. |
| EBTA-003 | Codex native usage events | Parse Codex `token_count` records into timestamped deltas, prefer `last_token_usage`, subtract cumulative `total_token_usage`, preserve model context, and honor `$CODEX_HOME/sessions` when no explicit directory is provided. | EBTA-001 | Yes, provider-events group | done | Codex | 2026-04-16 17:17 CST | `workspace/EBTA-003/` | `codex-native-usage-events` | Codex tests prove last usage, cumulative delta, cross-day timestamps, model context fallback, existing session parser compatibility, and `CODEX_HOME` source resolution; `go test ./provider/codex` passes. | Done. Verification passed: focused Codex event tests; `go test ./provider ./provider/codex`; `make fmt`; `make test`; `make vet`; `make build`; `make lint`. Review notes: `workspace/EBTA-003/review.md`. |
| EBTA-004 | Claude native usage events | Parse Claude assistant `message.usage` records into timestamped events using assistant timestamps and deduplicate streaming records by stable message/request keys. | EBTA-001 | Yes, provider-events group | todo | unclaimed | - | - | new | Claude tests prove assistant usage events, cross-day split inputs, streaming dedupe keeps latest usage, subagent paths remain included, and `go test ./provider/claude` passes. | First gate: `go test ./provider/claude -run TestParseClaudeUsageEvents`. Touches `provider/claude/parser.go`, `provider/claude/parser_test.go`. |
| EBTA-005 | Kimi native usage events | Parse Kimi `StatusUpdate` records with `token_usage` into timestamped incremental events while preserving metadata model/title/session behavior. | EBTA-001 | Yes, provider-events group | todo | unclaimed | - | - | new | Kimi tests prove each status update becomes one event with its own timestamp, cross-day records are preserved for aggregation, metadata fallback still populates model names, and `go test ./provider/kimi` passes. | First gate: `go test ./provider/kimi -run TestParseKimiUsageEvents`. Touches `provider/kimi/parser.go`, `provider/kimi/parser_test.go`. |
| EBTA-006 | Cursor event collector compatibility | Map Cursor CSV usage rows to native `UsageEvent` records without changing existing local CSV semantics or sync behavior. | EBTA-001 | Yes, provider-events group | todo | unclaimed | - | - | new | Cursor event tests prove CSV rows are emitted as timestamped usage events and existing Cursor parser/sync tests still pass with `go test ./provider/cursor ./cursor`. | First gate: `go test ./provider/cursor`. Touches `provider/cursor/parser.go`, `provider/cursor/parser_test.go` and only touches `cursor/` if shared helpers require it. |
| EBTA-007 | Daily command event aggregation | Switch `daily` from session-start aggregation to event-date aggregation, apply `--timezone`, preserve daily flag constraints, and keep JSON grouping semantics stable. | EBTA-002, EBTA-003, EBTA-004, EBTA-005, EBTA-006 | Yes, command-integration group | todo | unclaimed | - | - | new | Command tests prove same-session cross-day events split into the correct dates, `--timezone Asia/Shanghai` changes date keys, default rolling windows use local day boundaries, and `go test ./cmd -run TestDaily` passes. | First gate: `go test ./cmd -run TestDaily`. Touches `cmd/daily.go`, `cmd/daily_test.go`, possibly `cmd/help_text_test.go`. |
| EBTA-008 | Session command event aggregation | Switch `session` filtering to usage-event dates and group matching events by provider/session while preserving title, model, date, and token totals from included events. | EBTA-002, EBTA-003, EBTA-004, EBTA-005, EBTA-006 | Yes, command-integration group | todo | unclaimed | - | - | new | Command tests prove a session started before the filter is included when it has in-range usage events, totals include only filtered events, first/last included event dates are used, and `go test ./cmd -run TestSession` passes. | First gate: `go test ./cmd -run TestSession`. Touches `cmd/session.go`, `cmd/session_test.go`, possibly shared event aggregation helpers. |
| EBTA-009 | E2E cross-day acceptance fixtures | Add end-to-end fixtures and tests covering cross-day Codex cumulative usage, Claude assistant usage, Kimi status updates, daily JSON output, and session JSON output. | EBTA-007, EBTA-008 | No, acceptance after commands | todo | unclaimed | - | - | new | E2E tests prove `daily --json --since 2026-04-15 --until 2026-04-16 --timezone UTC` splits rows and `session --json --since 2026-04-16 --until 2026-04-16 --timezone UTC` includes only in-range event totals; targeted `go test ./e2e` passes. | First gate: `go test ./e2e -run Test`. Touches `e2e/e2e_test.go` and `e2e/testdata/` only as needed. |
| EBTA-010 | Docs, help, and final validation | Update user-facing docs/help for event-based aggregation, timezone behavior, and `CODEX_HOME`, then run full repository gates and manual CLI smoke checks. | EBTA-009 | No, final gate | todo | unclaimed | - | - | new | README and README_zh document event timestamps, session event filtering, local default timezone, `--timezone` IANA names, and `$CODEX_HOME/sessions`; `make fmt`, `make test`, `make vet`, `make lint` when available, `make build`, and documented CLI smoke checks pass or have recorded skip evidence. | First gate: `make fmt`. Touches `README.md`, `README_zh.md`, `docs/design.md` only if architecture docs need updating, and help text tests if command help changes. |

## Claiming Rules

- Before working, read this board and the source design.
- Claim exactly one `todo` task whose hard dependencies are all `done`.
- Update the task row first: set `Status` to `claimed`, set `Owner`, set `Claimed At`, and set `Workspace` to `workspace/<task-id>/`.
- Create the workspace directory after updating the row.
- Append a `Change Log` entry describing the claim.
- Do not edit another active task's workspace or revert unrelated repository changes.
- If implementation reveals the task split is wrong, update this board and append a `Change Log` entry before continuing.
- A task can move to `done` only when its `Done When` checks are satisfied and the relevant validation output is recorded in the task notes, workspace notes, or commit message.

## Change Log

- 2026-04-16 15:36 CST: Initialized task board from `docs/plans/2026-04-16-event-based-token-aggregation.md`; split the five design phases into ten claimable tasks covering core event aggregation, provider events, Cursor compatibility, command integration, e2e acceptance, docs, and final gates.
- 2026-04-16 15:53 CST: Codex claimed `EBTA-001` in worktree `.worktrees/ebta-001-core-events` with workspace `workspace/EBTA-001/`.
- 2026-04-16 15:53 CST: `EBTA-001` moved to `research` after workspace creation.
- 2026-04-16 15:58 CST: `EBTA-001` moved to `spec`; linked OpenSpec change `event-based-token-aggregation-core`.
- 2026-04-16 15:58 CST: `EBTA-001` moved to `implementing` after workspace plan, test strategy, and OpenSpec artifacts were ready.
- 2026-04-16 16:01 CST: `EBTA-001` moved to `verifying`; focused stats/provider tests passed.
- 2026-04-16 16:06 CST: `EBTA-001` moved to `review`; full gates passed including `make lint`.
- 2026-04-16 16:08 CST: `EBTA-001` moved to `done`; local review found no must-fix issues, and OpenSpec change remains active with all tasks checked for follow-up board work.
- 2026-04-16 17:17 CST: Codex claimed `EBTA-003` in worktree `.worktrees/ebta-003-codex-events` with workspace `workspace/EBTA-003/`.
- 2026-04-16 17:18 CST: `EBTA-003` moved to `research` after workspace creation and clean Codex provider baseline test.
- 2026-04-16 17:28 CST: `EBTA-003` moved to `spec`; linked OpenSpec change `codex-native-usage-events` after multi-agent review approved `final_impl_v1.md` and `test_strategy.md`.
- 2026-04-16 17:31 CST: `EBTA-003` moved to `implementing`; OpenSpec apply instructions show 0/7 tasks complete and ready for TDD.
- 2026-04-16 17:38 CST: `EBTA-003` moved to `verifying`; focused event tests and `go test ./provider/codex` passed.
- 2026-04-16 17:52 CST: `EBTA-003` moved to `review`; verification passed: `make fmt`, `make test`, `make vet`, `make build`, and `make lint`.
- 2026-04-16 17:59 CST: `EBTA-003` review found mixed last/total baseline, legacy session reset, and event collector parallelism gaps; fixes added and re-reviewed.
- 2026-04-16 18:03 CST: `EBTA-003` moved to `done`; re-review approved and fresh gates passed: `make fmt`, `make test`, `make vet`, `make build`, `make lint`.
