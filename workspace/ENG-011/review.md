# ENG-011 Review

Independent read-only review (subagent) of `runner-smoke.yml`, `docs/self-hosted-runner.md`, workspace and roadmap row. Scope check: `ci.yml` / `release.yml` untouched.

| Severity | Finding | Resolution |
| --- | --- | --- |
| must-fix | Doc presented the `head.repo.full_name` guard as the protection; for `pull_request` the workflow comes from the PR merge commit, so a fork can remove it. | Fixed: doc states the guard is hygiene; the boundary is the `all_external_contributors` approval policy plus never approving fork runs that touch workflows/Makefile/scripts. |
| should-fix | PATH loop `[ -d ] && echo` fails the step under `bash -e` if the last dir is missing. | Fixed: `if ...; then ...; fi`. |
| should-fix | Env step printed user name, HOME and PATH to world-readable logs. | Fixed: "Check tools" prints found/missing only and fails on missing. |
| should-fix | `workflow_run` rule vague; `push` not restricted. | Fixed: exact expressions and `branches: [main]` documented. |
| should-fix | `AI_REVIEW_PROXY` var not masked; smoke job lacked `NO_PROXY`. | Fixed: doc warning; smoke step exports `NO_PROXY`. |
| should-fix | Printing model answer allows `::` workflow-command injection. | Fixed: prints PASS/FAIL only. |
| nit | Branch behind main → ai-review.sh "no commits between". | Fixed: base = `merge-base origin/main HEAD`, fallback `HEAD~1`. |
| nit | `actions/checkout@v4` tag pin. | Fixed: pinned to SHA `11d5960…` (v4). |
| nit | `.path` is the config shell's PATH, not necessarily a login shell's. | Fixed wording. |
| nit | Public doc named the Mac's FileVault/auto-login setup. | Generalized. |

Deferred (see `todo.md`): CODEOWNERS / required review for `.github/workflows/`.

After fixes: the agy step script was re-run under `env -i` (minimal env) → `agy call: PASS`.
