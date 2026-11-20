# ENG-013 test strategy

What must be proven, and how:

| Claim | Proof |
| --- | --- |
| Candidate list is deterministic and ordered | `TestCollectIsSortedByKindAndDeterministic` (same input twice → same bytes; kind order; coverage ascending) |
| Each fixed tool is parsed correctly, excluded paths skipped | `TestParseLint…`, `TestParseDeadcode…`, `TestParseCoverage…`, `TestCollectDocDrift` (stale flag, undocumented flag, test-only/lookup flags ignored, placeholders ignored) |
| Exactly one candidate, rejected/abandoned ones skipped, empty → skip | `TestSelectSkipsExcludedAndRejectedCandidates`, `TestSelectOnEmptyListSkips` |
| One package / README-only scope and diff cap enforced | `TestCheckDiff` table (15 cases), `TestCheckCommandSkipsOnViolation` |
| At most 1 open optimization PR | `TestGateHonorsOpenPRLimit` |
| At most 2 fix rounds | `TestRoundBudget` |
| Review gate | `TestReviewVerdict` (pass with nits, Important items, prose finding, "No", missing verdict) |
| PR marker round-trips into rejection memory | `TestDescribeAndMarker` |
| Glue works with real tools | `scripts/auto-optimize.sh --dry-run` on this worktree; `--no-pr` rehearsal in a throwaway worktree with real acpx/agy/ai-review (results in `verification.md`) |
| Workflow on the runner | Manual `workflow_dispatch` after ENG-011 is merged and the runner is online (`todo.md`) |

Gates: `make fmt`, `make vet`, `make test`, `make build`, `make lint`.
