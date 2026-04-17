# EAP-007 Proposed Acceptance Flow

## Scope

EAP-007 is documentation and verification only unless final acceptance uncovers a behavior bug. No production code change is planned.

No OpenSpec change is needed because this task does not change CLI output, flag semantics, provider contracts, JSON shape, or local-only behavior. If verification exposes a behavior change, stop and handle that as a separate implementation task.

## Workflow

1. Record current implementation and existing guardrails in `workspace/EAP-007/`.
2. Synthesize a final implementation plan and test strategy.
3. Move the task board through `spec`, `implementing`, `verifying`, `review`, and `done`.
4. Run focused correctness checks before the broader gates.
5. Build `./bin/codetok` before manual CLI smoke or timing checks.
6. Record final acceptance evidence in the design plan and task board.

## Manual Timing Plan

Use the same local dataset family as the original baseline and measure the built binary. Prefer simple `/usr/bin/time -p` loops so results are easy to copy into task evidence.

Commands to time:

- `./bin/codetok daily`
- `./bin/codetok daily --json`
- `./bin/codetok daily --all --json`
- `./bin/codetok daily --provider claude --json`
- `./bin/codetok session --json`

For the default `daily` command, run at least three warm-cache iterations and record the median wall time. Compare it with the recorded 5.07s/5.26s baseline.

## Documentation Updates

Update:

- `docs/plans/2026-04-17-event-aggregation-performance-optimization.md` with final acceptance evidence and residual risk notes.
- `docs/plans/2026-04-17-event-aggregation-performance-optimization-task.md` with task status, verification evidence, and skipped conditional task rationale.

Do not update user-facing README files unless acceptance discovers a user-facing contract change.

## Residual-Risk Wording

Document EAP-004 and EAP-006 as included in final acceptance:

- EAP-004 is done on `origin/main` and should be included in the final dependency and evidence accounting.
- EAP-006 is done on `origin/main`; it required no additional code because EAP-004 already removed the materialized daily filter/aggregate path.

Do not leave a deferred EAP task in the final acceptance record once EAP-006 is done.
