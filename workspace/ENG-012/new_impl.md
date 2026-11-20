# ENG-012 New Implementation (proposal)

Reuse Dachein/Frontier-iOS#98: development hands over through one PR comment; a separate post-merge workflow archives.

1. Handoff: `scripts/post-records-comment.sh --pr N --task ID paths... [--inline NAME=FILE]` renders a comment with marker `<!-- codetok-records:ID -->`, a bullet list of paths and optional collapsed inline blocks, and upserts it (PATCH when a marker comment exists). No commit, no PR head change.
2. Archive: `.github/workflows/archive-records.yml` on `push` to `main` and `workflow_dispatch`, owner only (`github.triggering_actor == github.repository_owner`), `runs-on: [self-hosted, codetok-local, ai]`, runs `scripts/archive-records.sh --commit $GITHUB_SHA` (or `--pr N` on dispatch). The first version used `pull_request: closed`; see final_impl.md for why it changed.
3. Deterministic parts in a Go tool `scripts/archive-records/` (package main, tested by `make test`): marker parsing, path allowlist, verbatim copy from the merge commit via `git ls-tree`/`cat-file`, manifest with an input digest, prompt building, AI-output extraction and schema validation, README rendering, atomic directory swap, commit message.
4. AI: acpx + `agy acp`, `--deny-all --no-terminal`, prompt contains everything; output is JSON between `BEGIN_ARCHIVE_SUMMARY` / `END_ARCHIVE_SUMMARY`.
5. Destination `docs/changes/<task-id>/` (see final_impl.md for the reasoning).
