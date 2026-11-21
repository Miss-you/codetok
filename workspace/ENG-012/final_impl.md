# ENG-012 Final Implementation

`final_impl_v1.md` is the approved plan. Review fixes folded in during implementation:

- A root merge commit has no parent; the diff for the prompt falls back to the empty tree (found by the end-to-end script test).
- AI review round 1 (agy, head ebda22d): the marker now counts only on a line of its own (quoting it inline in a review no longer breaks the archive or the helper); `post-records-comment.sh` applies the same author_association filter as the archive, guards the empty-array case for macOS bash 3.2, checks `go`/`python3`, and requires listed paths to be tracked by git; a rejected summary prints the last 40 lines of agent output.
- AI review round 3 (head 8745401, verdict ready to merge): helper marker regex and whole-line check match the Go rule including trailing whitespace; `--inline FILE` resolves from the caller's directory; `--dry-run` checks acpx/agy before creating its directory; duplicate paths in the summary `files` list are rejected.
- AI review round 2 (head b139b25): the proxy is set before the first gh/git call; the helper rejects a directory without `/` or a file with `/` before posting; inline Python reads UTF-8 explicitly; the marker cannot span lines; the prompt fence is longer than any `~` run in its content.
- `truncateUTF8` trims only a rune cut by the limit, not the whole buffer, for binary-ish files.
- AI review round 4 (head 5884ccd, verdict ready to merge): minor findings only, deferred with reasons in the PR comment.
- AI review round 5 (head 518e0b9, after the push-trigger change, verdict with fixes): `actions/checkout` pinned by commit SHA and asserted for every `uses:` in `workflow_test.go`; agent output echoed on failure is prefixed with `  | ` so it cannot start a `::workflow-command::` (tested); `--inline` without `NAME=FILE` is rejected (tested); `git commit` runs with `commit.gpgsign=false`. The first commit's message no longer contains the literal skip-CI marker, so a squash merge message cannot skip the merge push. Deferred: `ARCHIVE_BRANCH` other than main (Plan only accepts main; not used), rebase-merge diff context, summary file coverage.

## Archive destination: `docs/changes/<task-id>/`, not `openspec/changes/archive/`

- Most tasks have no OpenSpec change (`Change=-` for every ENG task so far); their records are `workspace/` and task-board rows, which are not OpenSpec artifacts.
- `openspec/changes/archive/` is owned by the `openspec archive` CLI and the `openspec-archive-change` skill, which move a change directory there and sync specs. Writing other content into it would mix two producers and could collide with names it generates (`YYYY-MM-DD-<change>`).
- The archive must copy, not move: originals stay in place. OpenSpec archiving moves.
- `docs/changes/` matches the Frontier #98 design, so both repositories can share the format and a later cross-repo retrospective reads one layout.

When a task also has an OpenSpec change, the records comment lists `openspec/changes/<change>/`; the OpenSpec archive stays a separate, manual step.

## No OpenSpec change

`Change=-`: CI and devflow tooling only; no codetok CLI, provider or stats behavior changes.

## Limits

Per archived file 2 MiB, per PR 20 MiB; the prompt carries at most 48 KiB per file and 160 KiB of diff. The records comment must stay under GitHub's 65,536-character limit; larger content is committed under `workspace/<task-id>/` and listed by path.

## Trigger: push to main, owner only

The first version triggered on `pull_request: closed`. For `pull_request` events, `closed` included, GitHub runs the workflow file from the PR itself, so a fork PR could rewrite `archive-records.yml`, drop the `if:` guard, and run its own job on the self-hosted runner with the maintainer's local `gh`/`agy` credentials. Only the fork-PR approval policy (`all_external_contributors`, ENG-011) stood in the way, and it is unclear whether it covers `closed` events. Raised by the ENG-011 session and approved by the maintainer; changed as follows:

- `on: push: branches: [main]` plus `workflow_dispatch` (`pr` input). Both always run the workflow file from `main`. On push the script runs with `--commit $GITHUB_SHA` and finds the PR with `gh api repos/<repo>/commits/<sha>/pulls`, keeping only a PR whose `merge_commit_sha` is that sha, that is merged and whose base is `main`; exactly one must match. Anything else (a direct push, the archive commit itself) exits 0 with no commit.
- The job's `if:` adds `github.triggering_actor == github.repository_owner`, so jobs on this runner stay limited to the owner if collaborators are added later.
- Loop: the archive commit is itself a push to `main`. GitHub does not start push workflows for a `[skip ci]` commit; the job's `if:` checks `!contains(github.event.head_commit.message, '[skip ci]')` too; and if it ran anyway, the commit is not a PR merge (skipped) and the input digest makes it a no-op.
- The fork check stays in the script (`isCrossRepository`): a fork PR that the owner merges is not archived.

A fork PR can still add a new workflow file of its own that targets the runner; that is the general self-hosted risk covered by the fork-PR approval policy in `docs/self-hosted-runner.md`, not by this workflow.
