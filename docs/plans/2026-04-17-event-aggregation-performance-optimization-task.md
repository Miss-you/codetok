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
| EAP-002 | Parallelize Claude event parsing | Make Claude usage event collection parse files with bounded concurrency while preserving output totals and local-only behavior. | EAP-001 | provider | todo | - | - | - | TBD | `go test ./provider/claude` and focused daily/session tests pass; `daily --provider claude --json` totals match pre-change results; local timing improves materially. | First gate: `go test ./provider/claude`. |
| EAP-003 | Range-aware provider candidate filtering | Push date-window candidate selection into provider collection so default `daily` avoids provably inactive history while exact event timestamp filtering remains authoritative. | EAP-001 | provider,cmd,stats | todo | - | - | - | TBD | Default `daily` parses fewer inactive files on synthetic fixtures, cross-day sessions remain included, `--all` remains equivalent to full-history collection, and date flag semantics are unchanged. | First gate: focused package tests for provider candidate filtering plus `go test ./cmd -run 'TestRunDaily|TestRunSession'`. |
| EAP-004 | Streaming daily aggregation if needed | If range-aware materialized collection is still too expensive, aggregate selected events without building a full `[]UsageEvent` for daily. | EAP-002,EAP-003 | cmd,provider,stats | todo | - | - | - | TBD | Dashboard and JSON output match the materialized path, parser errors retain provider context, and pprof allocation evidence improves versus the range-aware materialized path. | Conditional; first gate: `go test ./cmd ./stats`. |
| EAP-005 | Reduce Codex JSON parse churn | Optimize Codex parsing allocations and CPU only after candidate filtering narrows the real workload. | EAP-001 | provider | todo | - | - | - | TBD | Existing Codex delta/model fallback tests pass and same-data CPU/allocation profiles show lower Codex JSON cost. | First gate: `go test ./provider/codex`. |
| EAP-006 | Collapse filter and aggregation pass | If events remain materialized, combine date filtering and daily aggregation as a cleanup optimization without changing JSON semantics. | EAP-003 | stats,cmd | todo | - | - | - | TBD | `stats` and daily command tests pass, distinct session counting and grouping semantics remain unchanged, and no user-facing JSON fields change. | First gate: `go test ./stats ./cmd -run TestRunDaily`. |
| EAP-007 | Final performance acceptance and docs | Re-run local timing, compare against baseline, update plan evidence, and document residual risks or skipped conditional tasks. | EAP-002,EAP-003 | docs,verification | todo | - | - | - | - (documentation and verification only unless new behavior is discovered) | Fresh `make fmt`, `make test`, `make vet`, optional `make lint`, `make build`, and manual CLI smoke results are recorded; residual tasks are explicit. | First gate: `make test`. |

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
