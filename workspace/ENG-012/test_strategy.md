# ENG-012 Test Strategy

All automated checks: `go test ./scripts/archive-records/` (included in `make test`).

| Behavior to prove | Test |
| --- | --- |
| Comment format round-trips (paths, inline blocks with tricky content, CRLF, ``` fences) | `TestRenderParseRoundTrip`, `TestParseCommentCRLFAndBacktickFence` |
| Malformed or dangerous comments fail loudly (two markers, bad id, `..`, absolute, outside allowlist, `docs/changes`, wildcards, pathspec magic, duplicates, broken fences) | `TestParseCommentErrors` |
| Untrusted authors' markers are ignored; two trusted comments is an error | `TestFindRecords` |
| Files copied byte for byte from the merge commit, directories expanded, overlap deduplicated | `TestPlanCopiesVerbatim` |
| Skip without side effects: fork, unmerged, other base, no marker, untrusted marker | `TestPlanSkips` |
| Missing path, file/dir mismatch, symlink fail | `TestPlanErrors` |
| Same inputs → skip; changed inputs → archive | `TestPlanIdempotentDigest` |
| AI output extraction and schema validation (15 rejection cases) | `TestExtractSummary`, `TestParseSummaryRejects` |
| Invalid AI output writes nothing; valid output replaces the old directory atomically | `TestFinalizeWritesOnlyValidOutput` |
| Commit message `docs: archive <id> records (#<pr>) [skip ci]` | `TestCommitMessageSkipsCI` |
| Workflow triggers only on pull_request/workflow_dispatch, fork guard, runner labels, no secrets, no `pull_request_target`/`issue_comment` | `TestArchiveWorkflowGuards` |
| Whole shell flow against a local bare origin with stub gh/acpx/agy: push with the right message, only `docs/changes/<id>/` touched, verbatim files, rerun makes no commit and no AI call, no-marker and fork PRs skip, invalid AI output fails with no commit, acpx called with `--deny-all --no-terminal`, temp worktree removed | `TestArchiveScriptEndToEnd` |

Manual (needs the runner from ENG-011 and a real merge): see `todo.md`.
