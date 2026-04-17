# Event Aggregation Performance Optimization Task Board

## Source Design

- Source plan: `docs/plans/2026-04-17-event-aggregation-performance-optimization.md`
- Goal: make `codetok daily` fast again after event-based aggregation while preserving event-timestamp attribution and cross-day session correctness.
- Non-goals: do not add durable parsed-event caches, do not make reporting commands call remote provider APIs, and do not use file metadata as the final attribution source.
- Primary implementation order: baseline first, then parser parallelism, then range-aware provider collection, then deeper parser or aggregation cleanup only if measurement still justifies it.

## Status Legend

- `todo`: ready only when all hard dependencies are `done`
- `claimed`: owner has reserved the task and created or is creating its workspace
- `research`: current implementation and risks are being documented
- `spec`: final approach and test strategy are being made executable
- `implementing`: code, tests, fixtures, or docs are being changed
- `verifying`: fresh validation is running and failures are being handled
- `review`: independent review is in progress or being addressed
- `blocked`: progress stopped; `Notes` must include the blocker and `resume_to=<state>`
- `done`: implementation, evidence, review, and task-board state are consistent

## Dependency Rules

- `EAP-001` must land first because later performance claims need stable correctness fixtures and repeatable measurement hooks.
- `EAP-002`, `EAP-003`, and `EAP-005` may proceed in parallel only after `EAP-001` is done, because they touch different bottleneck surfaces but share acceptance evidence.
- `EAP-004` is conditional and should start only if `EAP-002` plus `EAP-003` still leave materialization or memory as a measured bottleneck.
- `EAP-006` is cleanup-only and should start after parser and candidate filtering work, because current evidence shows filtering and aggregation are not the primary bottleneck.
- `EAP-007` closes the overall plan and depends on all implemented optimization tasks.

## Task Table

| ID | Title | Goal | Depends On | Parallel | Status | Owner | Claimed At | Workspace | Change | Done When | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| EAP-001 | Baseline fixtures and benchmarks | Add reusable correctness fixtures and repeatable baseline benchmarks before parser behavior changes. | - | no | done | Codex | 2026-04-17 12:06:38 CST | `workspace/EAP-001/` | - (test/tooling only; no OpenSpec because product behavior is unchanged) | Existing correctness tests pass, new fixtures protect event timestamp attribution and cross-day sessions, and a benchmark/reportable harness exposes considered/parsed/emitted/filter counts or the closest currently available baseline. | Verified with focused provider/cmd/stats/e2e checks, `make fmt`, `make test`, `make vet`, `make lint`, and `make build`. |
| EAP-002 | Parallelize Claude event parsing | Make Claude usage event collection parse files with bounded concurrency while preserving output totals and local-only behavior. | EAP-001 | provider | done | Codex | 2026-04-17 13:34:46 CST | `workspace/EAP-002/` | - (provider performance only; no OpenSpec because user-visible behavior and specs are unchanged) | `go test ./provider/claude` and focused daily/session tests pass; `daily --provider claude --json` totals match pre-change results; local timing improves materially. | Final gates passed: focused provider/cmd tests, `make fmt`, `make test`, `make vet`, `make lint`, `make build`; baseline/current Claude JSON matched with current 1.64s vs baseline 3.16s. |
| EAP-003 | Range-aware provider candidate filtering | Push date-window candidate selection into provider collection so default `daily` avoids provably inactive history while exact event timestamp filtering remains authoritative. | EAP-001 | provider,cmd,stats | done | Codex | 2026-04-17 13:35:53 CST | `workspace/EAP-003/` | - (internal performance optimization; user-facing CLI output and date contracts preserved) | Default `daily` parses fewer inactive files on synthetic fixtures, cross-day sessions remain included, `--all` remains equivalent to full-history collection, and date flag semantics are unchanged. | Verified with focused command/provider tests, cross-day e2e, `make fmt`, `make test`, `make vet`, `make lint`, `make build`, and built-binary smoke checks. |
| EAP-004 | Streaming daily aggregation if needed | If range-aware materialized collection is still too expensive, aggregate selected events without building a full `[]UsageEvent` for daily. | EAP-002,EAP-003 | cmd,provider,stats | done | Codex | 2026-04-17 15:07:13 CST | `workspace/EAP-004/` | - (internal performance optimization; daily CLI output and JSON contracts preserved) | Dashboard and JSON output match the materialized path, parser errors retain provider context, and allocation benchmark improves versus the range-aware materialized command path. | Verified with focused tests, provider/e2e checks, synthetic benchmem evidence, `make fmt`, `make test`, `make vet`, `make lint`, `make build`, and built-binary smoke checks. |
| EAP-005 | Reduce Codex JSON parse churn | Optimize Codex parsing allocations and CPU only after candidate filtering narrows the real workload. | EAP-001 | provider | done | Codex | 2026-04-17 13:31:30 CST | `workspace/EAP-005/` | - (provider-only parser performance; no user-visible contract change) | Existing Codex delta/model fallback tests pass and same-data CPU/allocation profiles show lower Codex JSON cost. | Verified with focused Codex tests/benchmark, no-must-fix agent re-review, `make fmt`, uncached `make test`, `make vet`, `make lint`, and `make build`. |
| EAP-006 | Collapse filter and aggregation pass | If events remain materialized, combine date filtering and daily aggregation as a cleanup optimization without changing JSON semantics. | EAP-003 | stats,cmd | done | Codex | 2026-04-17 16:41:30 CST | `workspace/EAP-006/` | - (no code change; EAP-004 streaming daily aggregation already removed the materialized filter/aggregate pass while preserving contracts) | `stats` and daily command tests pass, distinct session counting and grouping semantics remain unchanged, and no user-facing JSON fields change. | Closed after rebasing onto `origin/main`: EAP-004 already streams range filtering plus aggregation through `aggregateDailyUsageEventsFromProvidersInRange`, so no additional EAP-006 code is needed; fresh verification rerun before PR. |
| EAP-007 | Final performance acceptance and docs | Re-run local timing, compare against baseline, update plan evidence, and document residual risks or skipped conditional tasks. | EAP-002,EAP-003 | docs,verification | todo | - | - | - | - (documentation and verification only unless new behavior is discovered) | Fresh `make fmt`, `make test`, `make vet`, optional `make lint`, `make build`, and manual CLI smoke results are recorded; residual tasks are explicit. | First gate: `make test`. |
| EAP-007 | Final performance acceptance and docs | Re-run local timing, compare against baseline, update plan evidence, and document residual risks or skipped conditional tasks. | EAP-001,EAP-002,EAP-003,EAP-004,EAP-005,EAP-006 | docs,verification | done | Codex | 2026-04-17 16:21:19 CST | `workspace/EAP-007/` | - (documentation and verification only unless new behavior is discovered) | Fresh `make fmt`, `make test`, `make vet`, optional `make lint`, `make build`, manual CLI smoke/timing results, and metric evidence are recorded; residual tasks are explicit. | Final verification and independent review passed. After rebasing onto EAP-004 and EAP-006, default `daily` median was 0.51s versus 5.07s/5.26s baseline; no EAP optimization tasks remain open. |

## Claiming Rules

- Only claim a task with `Status=todo` and all hard dependencies marked `done`.
- Update this board before starting work: set `Owner`, `Claimed At`, `Workspace`, and append to `Change Log`.
- Create `workspace/<task-id>/` inside the active worktree before research or implementation.
- If a task changes CLI behavior or provider/stat semantics, record the OpenSpec change ID in `Change`; if it is test/tooling-only, explain why `Change=-` is acceptable.
- Keep each task scoped to its `Goal`; broader findings go into that task's workspace `todo.md` or a later task row.
- Move statuses incrementally through `research`, `spec`, `implementing`, `verifying`, `review`, and `done`; do not jump straight from `claimed` to `done`.

## Change Log

- 2026-04-17 12:06:38 CST: Initialized task board from `docs/plans/2026-04-17-event-aggregation-performance-optimization.md`.
- 2026-04-17 12:06:38 CST: Claimed `EAP-001` for Codex in `workspace/EAP-001/` and moved it to `research`.
- 2026-04-17 12:12:00 CST: Completed research/spec workspace notes for `EAP-001`; no OpenSpec needed because the task is test/benchmark-only; moved to `implementing`.
- 2026-04-17 12:18:00 CST: Added provider-level benchmark and cross-day mtime guardrail; moved `EAP-001` to `verifying`.
- 2026-04-17 12:18:30 CST: Requested independent agent review and moved `EAP-001` to `review`.
- 2026-04-17 12:19:04 CST: Addressed review feedback by adding explicit date-window filtering to the Codex guardrail; fresh verification passed; moved `EAP-001` to `done`.
- 2026-04-17 13:31:30 CST: Claimed `EAP-005` for Codex in `workspace/EAP-005/` and moved it to `research`.
- 2026-04-17 13:34:46 CST: Claimed `EAP-002` for Codex in `workspace/EAP-002/` and moved it to `research`.
- 2026-04-17 13:35:53 CST: Claimed `EAP-003` for Codex in `workspace/EAP-003/`.
- 2026-04-17 13:35:53 CST: Moved `EAP-003` to `research`; baseline focused checks passed (`go test ./provider/...`, `go test ./cmd -run 'TestRunDaily|TestRunSession'`, `go test ./stats`). Full `go test ./...` baseline was interrupted after e2e stalled under concurrent EAP worktrees.
- 2026-04-17 13:39:05 CST: Completed research/spec workspace notes for `EAP-002`; no OpenSpec needed because behavior is unchanged; moved to `implementing`.
- 2026-04-17 13:40:40 CST: Focused RED/GREEN provider tests passed and independent review found no must-fix issues; moved `EAP-002` to `verifying`.
- 2026-04-17 13:46:00 CST: Completed multi-agent research/spec review for `EAP-005`; no OpenSpec needed because behavior and CLI contracts are unchanged; moved to `implementing`.
- 2026-04-17 13:46:18 CST: Completed multi-agent research notes and final implementation/test strategy artifacts; no OpenSpec change because user-facing CLI output and date contracts are preserved; moved `EAP-003` to `spec`.
- 2026-04-17 13:46:33 CST: Final verification passed and Claude JSON matched the EAP-001 baseline; moved `EAP-002` to `review`.
- 2026-04-17 13:46:35 CST: Began TDD implementation for `EAP-003`; moved to `implementing`.
- 2026-04-17 13:46:55 CST: Closed `EAP-002` as `done`; no deferred follow-ups beyond later EAP range/filtering tasks already on the board.
- 2026-04-17 13:55:00 CST: Added Codex typed model extraction, parser hot-path guardrails, and benchmark evidence; moved `EAP-005` to `verifying`.
- 2026-04-17 14:08:27 CST: Addressed pre-implementation review findings, completed range-aware provider collection, and verified with focused tests, full `make test`, `make vet`, `make lint`, `make build`, and CLI smoke checks; moved `EAP-003` to `done`.
- 2026-04-17 14:22:00 CST: Addressed agent review findings for model propagation precedence and skipped token counts; fresh verification passed; moved `EAP-005` to `done`.
- 2026-04-17 15:07:13 CST: Claimed `EAP-004` for Codex in `workspace/EAP-004/`.
- 2026-04-17 15:07:35 CST: Moved `EAP-004` to `research`; using multi-agent review to confirm whether streaming aggregation is still justified after range-aware collection and to define the narrowest implementation.
- 2026-04-17 15:15:00 CST: Completed multi-agent research and final implementation/test strategy artifacts; no OpenSpec change because user-visible daily behavior is unchanged; moved `EAP-004` to `spec`.
- 2026-04-17 15:18:00 CST: Added RED tests for stats incremental aggregation and direct daily aggregation, then implemented the shallow streaming daily path; moved `EAP-004` to `implementing`.
- 2026-04-17 15:23:00 CST: Focused RED/GREEN checks passed for stats incremental aggregation and daily JSON/dashboard parity; moved `EAP-004` to `verifying`.
- 2026-04-17 15:35:00 CST: Verification passed (`make fmt`, `make test`, `make vet`, `make lint`, `make build`, focused tests, static daily no-allEvents check, and built-binary smoke); moved `EAP-004` to `review`.
- 2026-04-17 15:38:00 CST: Review recorded no must-fix issues; closed `EAP-004` as `done`.
- 2026-04-17 16:13:24 CST: Addressed Copilot PR review comments by normalizing the aggregator on every add, reusing a date-range filter in hot paths, and restoring bulk append in the materialized adapter.
- 2026-04-17 16:41:30 CST: Reviewed `EAP-006` after rebasing onto latest `origin/main`; EAP-004 had already replaced the materialized daily event path with streaming range filtering plus aggregation, so EAP-006 required no additional code change; moved `EAP-006` to `done` pending fresh verification and PR.
- 2026-04-17 16:21:19 CST: Claimed `EAP-007` for Codex in `workspace/EAP-007/`; isolated worktree baseline `make test` passed and task moved to `research`.
- 2026-04-17 16:30:15 CST: Completed multi-agent research and planning review for `EAP-007`; no OpenSpec change required because this is verification/documentation only; moved to `spec`.
- 2026-04-17 16:31:03 CST: Final test strategy and implementation approach recorded for `EAP-007`; moved to `implementing`.
- 2026-04-17 16:31:29 CST: Started final verification for `EAP-007`; `golangci-lint` is installed so `make lint` is in scope.
- 2026-04-17 16:41:13 CST: Recorded final verification and timing evidence for `EAP-007`; moved to `review`.
- 2026-04-17 16:42:55 CST: Independent final review accepted the EAP-007 closeout packet; moved `EAP-007` to `done`.
- 2026-04-17 16:54:55 CST: Rebased `EAP-007` onto `origin/main` after `EAP-004` landed, resolved task-board conflict, reran verification/timing, and refreshed final acceptance evidence.
- 2026-04-17 17:03:00 CST: Rebased `EAP-007` onto `origin/main` after `EAP-006` landed; refreshed closeout wording so all EAP tasks are accounted for as done.
