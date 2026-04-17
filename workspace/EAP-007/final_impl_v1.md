# EAP-007 Final Implementation V1

## Decision

Complete EAP-007 as a verification and documentation task. Do not change production code unless final checks expose a real behavior regression.

## Implementation Steps

1. Keep task board status synchronized at each phase.
2. Record research and final acceptance artifacts under `workspace/EAP-007/`.
3. Mark `Change=-` because no OpenSpec artifact is required for documentation-only acceptance.
4. Run focused checks for the optimized path:
   - command range wiring and date semantics
   - provider range-aware candidate filtering
   - stats event timestamp attribution
   - cross-day e2e behavior
   - explicit `session --since/--until` behavior
   - provider metric assertions for considered/skipped/parsed/emitted counts
5. Capture `workspace/EAP-007/verification.md` as the durable evidence log.
6. Run full gates:
   - `make fmt`
   - `make test`
   - `make vet`
   - `make lint` when `golangci-lint` is installed
   - `make build`
7. Run built-binary smoke and timing checks for `daily`, `daily --json`, `daily --all --json`, provider-specific `daily`, `session --json`, and `session --since/--until --json`.
8. Append final acceptance evidence to the design plan.
9. Update task board notes and close EAP-007.

## Acceptance Interpretation

The original baseline was about 5.1s to 5.3s for default `daily` on the local dataset. A materially faster final timing supports closing EAP-007 only when paired with correctness gates and metric evidence showing that range-aware provider collection skips inactive candidates while preserving exact event timestamp filtering.

Local timing is supporting evidence, not a deterministic test. The hard pass/fail gates remain correctness tests, race-covered `make test`, vet, build, lint when available, and built-binary smoke checks.

EAP-004 and EAP-006 are included in the final acceptance pass once rebased onto `origin/main`; no EAP optimization tasks remain open.

## Out Of Scope

- Durable parsed-event caches
- Remote provider API calls from reporting commands
- User-facing CLI or JSON contract changes
- Streaming aggregation unless final evidence makes it necessary
- Collapsing stats filtering and aggregation unless final evidence makes it necessary
