# docs/changes: archived process records

This directory holds the process records of merged task PRs, one directory per task id (`ENG-012`, `EBTA-003`, an OpenSpec change name, ...). It is written by `.github/workflows/archive-records.yml` after a PR merges into `main` ([#57](https://github.com/Miss-you/codetok/issues/57)). Nobody edits it by hand.

## How a directory gets here

1. While delivering a task, the agent posts one PR comment marked `<!-- codetok-records:<task-id> -->` with `scripts/post-records-comment.sh`. It lists the process files (`workspace/<task-id>/`, `openspec/changes/<change>/`, `docs/plans/...`) and may carry uncommitted notes in collapsed blocks. Rerunning the script edits that comment; it never adds a second one.
2. When the PR merges, the workflow runs `scripts/archive-records.sh --pr <number>` on the self-hosted runner (`runs-on: [self-hosted, codetok-local, ai]`).
3. The script copies the listed files byte for byte from the merge commit into `<task-id>/raw/`. agy (through acpx, read-only) only writes a summary in a fixed JSON schema; the script validates it and renders `README.md`. Invalid output fails the run and writes nothing.
4. One commit lands on `main`: `docs: archive <task-id> records (#<PR>) [skip ci]`.

PRs without a records comment, PRs from forks, and comments from users who are not owners, members or collaborators are skipped.

## Layout

```
docs/changes/<task-id>/
  README.md       summary rendered from summary.json, with links into raw/
  summary.json    the AI summary: what changed, decisions and reasons, follow-ups, file roles
  manifest.json   PR number, title, URL, merge commit, records comment id, file list with sha256, input_digest
  raw/<path>      verbatim copy of each listed repository file as of the merge commit
  raw/inline/<name>  verbatim copy of each inline block from the comment
```

## Rules

- These are history. Do not edit them to match later code; the next PR for the same task replaces the directory, and `git log -- docs/changes/<task-id>` keeps earlier versions.
- Rerunning the workflow for the same PR is a no-op: when `manifest.json` already has the same `input_digest`, the script stops before calling the AI and makes no commit.
- No credentials, tokens or screenshots. The records comment is public before it is archived; check it before posting.
- The originals under `workspace/`, `openspec/changes/` and `docs/plans/` stay where they are. The archive never moves or rewrites them.

## Searching across tasks

```bash
# Decisions recorded for every task
grep -h -A1 '^## Key decisions' docs/changes/*/README.md
# Which tasks left open follow-ups
grep -l '\[todo\]\|\[handoff\]\|\[unverified\]' docs/changes/*/README.md
# Which PR archived a task, and from which merge commit
python3 -c 'import json,glob; [print(m["task_id"], m["pr"]["number"], m["pr"]["merge_commit"][:12]) for m in map(lambda p: json.load(open(p)), glob.glob("docs/changes/*/manifest.json"))]'
```

## Running it by hand

From a clean checkout with `gh`, `go`, `acpx` and `agy` on `PATH`:

```bash
scripts/archive-records.sh --pr <number> --dry-run   # stage files and the prompt, no AI call
scripts/archive-records.sh --pr <number> --no-push   # commit in a temporary worktree, do not push
```

Tests for the deterministic parts: `go test ./scripts/archive-records/` (also part of `make test`).
