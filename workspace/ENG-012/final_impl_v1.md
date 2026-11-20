# ENG-012 Final Implementation v1

Adopts new_impl.md with these decisions:

- Trust: on a public repo anyone can comment, so only marker comments whose `author_association` is OWNER, MEMBER or COLLABORATOR count; others are logged and ignored. More than one trusted marker comment is an error (the helper keeps one).
- Paths: only `workspace/`, `openspec/changes/`, `docs/plans/`; clean relative paths, no `..`, `.`, wildcards, backslashes or pathspec magic; read with `git --literal-pathspecs`. Symlinks and submodules are rejected. A listed path missing at the merge commit fails the run (the comment and the archive must agree).
- Source of truth for file content is the merge commit, read from git objects, never the work tree.
- Idempotency: `input_digest` = sha256 over task id, PR number and (source, sha256, path) of every archived file. If `docs/changes/<id>/manifest.json` on main has the same digest, stop before the AI call; no commit. Otherwise overwrite the directory (stale files removed). A final `git diff --cached --quiet` guard also prevents empty commits.
- Invalid AI output: extraction or validation failure exits non-zero before anything is written to the repo.
- Writes happen in a temporary `git worktree` at `origin/main`, so the runner checkout or a developer tree is untouched. Push `HEAD:main` with the runner user's credentials (`persist-credentials: false`), rebase-and-retry up to 3 times; workflow `concurrency` serializes runs.
- Loop prevention: (a) the workflow has no `push` trigger; (b) the commit message ends in `[skip ci]`, so push-triggered workflows (CI, any later AI review workflow) skip it; (c) the archive never opens a PR, and `scripts/ai-review.sh` only reviews PR heads. (a) and (b) are asserted by tests.
- Fork PRs: job `if` checks `head.repo.full_name == github.repository` and `github.repository == 'Miss-you/codetok'`; the script checks `isCrossRepository` again.
