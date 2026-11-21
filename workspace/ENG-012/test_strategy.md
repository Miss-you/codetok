# ENG-012 Test Strategy

All automated checks: `go test ./scripts/archive-records/` (included in `make test`).

| Behavior to prove | Test |
| --- | --- |
| Comment format round-trips (paths, inline blocks with tricky content, CRLF, ``` fences) | `TestRenderParseRoundTrip`, `TestParseCommentCRLFAndBacktickFence` |
| Malformed or dangerous comments fail loudly (two markers, bad id, `..`, absolute, outside allowlist, `docs/changes`, wildcards, pathspec magic, duplicates, broken fences) | `TestParseCommentErrors` |
| A marker quoted inline in another comment is not a records comment | `TestInlineMarkerMentionIsNotARecord` |
| Untrusted authors' markers are ignored; two trusted comments is an error | `TestFindRecords` |
| Files copied byte for byte from the merge commit, directories expanded, overlap deduplicated | `TestPlanCopiesVerbatim` |
| Skip without side effects: fork, unmerged, other base, no marker, untrusted marker | `TestPlanSkips` |
| Missing path, file/dir mismatch, symlink fail | `TestPlanErrors` |
| Same inputs → skip; changed inputs → archive | `TestPlanIdempotentDigest` |
| AI output extraction and schema validation (15 rejection cases) | `TestExtractSummary`, `TestParseSummaryRejects` |
| Invalid AI output writes nothing; valid output replaces the old directory atomically | `TestFinalizeWritesOnlyValidOutput` |
| Commit message `docs: archive <id> records (#<pr>) [skip ci]` | `TestCommitMessageSkipsCI` |
| Workflow triggers only on push to main and workflow_dispatch (no `pull_request`, `pull_request_target`, `issue_comment`), owner-only `triggering_actor` guard, `[skip ci]` guard, `--commit "$GITHUB_SHA"`, runner labels, no secrets | `TestArchiveWorkflowGuards` |
| Whole shell flow against a local bare origin with stub gh/acpx/agy: push with the right message, only `docs/changes/<id>/` touched, verbatim files, rerun makes no commit and no AI call, push sha resolves to its merged PR, pushes that are not a PR merge (direct push, open PR, other base, other sha) skip, bad `--commit` input rejected, no-marker and fork PRs skip, invalid AI output fails with no commit, acpx called with `--deny-all --no-terminal`, temp worktree removed | `TestArchiveScriptEndToEnd` |
| Helper script: no paths, dir without `/`, file with `/`, untracked path fail before any gh call; POST when none, PATCH when one, refuse another task's or two comments; posted body parses | `TestPostRecordsCommentScript` |

Manual (needs the runner from ENG-011 and a real merge): see `todo.md`.
