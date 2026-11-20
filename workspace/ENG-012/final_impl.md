# ENG-012 Final Implementation

`final_impl_v1.md` is the approved plan. Review fixes folded in during implementation:

- A root merge commit has no parent; the diff for the prompt falls back to the empty tree (found by the end-to-end script test).
- AI review round 1 (agy, head ebda22d): the marker now counts only on a line of its own (quoting it inline in a review no longer breaks the archive or the helper); `post-records-comment.sh` applies the same author_association filter as the archive, guards the empty-array case for macOS bash 3.2, checks `go`/`python3`, and requires listed paths to be tracked by git; a rejected summary prints the last 40 lines of agent output.
- AI review round 3 (head 8745401, verdict ready to merge): helper marker regex and whole-line check match the Go rule including trailing whitespace; `--inline FILE` resolves from the caller's directory; `--dry-run` checks acpx/agy before creating its directory; duplicate paths in the summary `files` list are rejected.
- AI review round 2 (head b139b25): the proxy is set before the first gh/git call; the helper rejects a directory without `/` or a file with `/` before posting; inline Python reads UTF-8 explicitly; the marker cannot span lines; the prompt fence is longer than any `~` run in its content.
- `truncateUTF8` trims only a rune cut by the limit, not the whole buffer, for binary-ish files.

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

## Known risk outside this task

A fork PR can modify workflow files in its own branch; whether such a run reaches the self-hosted runner depends on the repository's fork-PR approval policy (set to `all_external_contributors` by ENG-011). This workflow's own guards do not cover that case.
