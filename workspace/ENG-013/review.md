# ENG-013 review

## Independent code review (subagent, fresh context, read-only)

7 findings, all addressed in `fix: address independent review of auto-optimize tooling`:

| # | Severity | Finding | Action |
| --- | --- | --- | --- |
| 1 | High | `verify` ran `make test`/`make lint` even when the scope check failed, so out-of-scope agent code (e.g. Makefile) would execute | Scope check runs first; on failure tests are not run. Tests run under `sandboxed` (empty HOME, no proxy/tokens, `GOPROXY=off`). Docs state that agent-written test code still runs as the runner user. |
| 2 | High | Agent edit permission could touch `.git/` (hooks, config, excludes), ignored files, or the run's own state | `guard_snapshot`/`guard_check` around every agent turn: hash git config/hooks/info, ignored-file list, tool binary, candidate, round counter; on change restore git files, delete new ignored files, end run. Default out dir moved outside the checkout. |
| 3 | High | Verdict parser missed bold section titles, reset on `#` lines inside sections, and treated "None of the callers…" as "None" | Bold-only lines are titles; unknown headings stay in the section; code fences ignored; "none" must be the whole line; Critical and Important sections must both be present. Four new fixtures. |
| 4 | Medium | Agent text and the raw review trace were posted publicly | PR body no longer quotes the agent; comment carries only the reviewer's answer; `--suppress-reads` on the agent; token-pattern scan before publishing; artifact limited to reports. |
| 5 | Medium | Infra failures permanently abandoned a candidate | `give_up` (no record) for agent/reviewer/guard failures; `abandon` (recorded) only for check, scope, review or empty-change outcomes. |
| 6 | Low | `--no-pr` left a branch that made a same-day rerun fail | `git checkout -B`. |
| 7 | Low | `git add`/`numstat` failures inside `verify` were silent | Explicit `die` on failure; empty-diff check moved before the scope check. |

## AI review of the PR

Run `scripts/ai-review.sh --pr <number>` after opening the PR, per AGENTS.md; result goes in an `AI review: <sha>` PR comment.

### PR #62, head b9e0a8b (`scripts/ai-review.sh --pr 62`, gemini-4-hydrogen)

Critical: none. "Ready to merge? With fixes". All findings verified and fixed in the next commit:

| Finding | Level | Action |
| --- | --- | --- |
| `guard_state` skipped ignored files under `output/`, `bin/`; `git status` quoted paths | Important | Exclusion removed (only new paths are acted on); `core.quotePath=false`. |
| Same-day branch name collides after a push-then-`gh pr create` failure | Important | Branch is `auto-optimize/<UTC time>-<id>`. |
| `review_answer` ignored bold-title reviews; empty output could fail under `pipefail` | Important | Starts at a heading or bold title line; filtering done inside `awk`. |
| Tool built before checking out the base | Minor | Rebuilt right after `git checkout --detach $base`. |
| `gate -max-open 0` panics | Minor | Usage error for values below 1; test added. |
| Closed-PR history capped at 200; root-package `testdata/` | Minor | `--limit 1000`; `path.Join`; test added. |
| Stale `rounds` in a reused out dir; any `round` error abandoned the candidate | Minor | `rounds` cleared at start; only exit 10 abandons, other errors `die`. |
| Recommendation: `deadcode@v0.52.0` download needs the proxy | — | `setup_proxy` now runs before candidate collection in PR and no-pr modes. |

### PR #62, head 75e84e8 (round 2)

Critical: none. "Ready to merge? With fixes". Fixed in the next commit:

| Finding | Level | Action |
| --- | --- | --- |
| Fix-round prompt lacked the candidate and kind guidance (each `acpx exec` is a fresh session) | Important | `candidate_block` included in the fix prompt. |
| Agent cannot delete an out-of-scope new file, so fix rounds could never recover | Important | `check -out-of-scope` lists offending paths; `revert_out_of_scope` restores them from base or deletes new ones before the fix round; test added. |
| `[[ … ]] && rm` at the end of the guard cleanup loop could trip `set -e` | Important | `if … then … fi`. |
| Proxy set after the first `gh`/`git` network calls | Minor | `setup_proxy` right after preflight in PR/no-pr modes. |
| Any `verdict` error abandoned the candidate | Minor | Only exit 10 abandons; other errors `give_up`. |
| Reused out dir kept stale reports | Minor | Per-run files cleared at start. |
| `gofmt` file list without `core.quotePath=false` | Minor | Added. |
| `sectionFor` matched titles that only mention "critical" | Minor | Titles must start with the severity; fixture added. |
| `-max-lines`/`-max` not validated; root `testdata/` lint; `AUTO_OPTIMIZE_BASE` undocumented | Minor | Validated with tests; skipped; documented. `collectDocDrift` stays non-recursive: `cmd/` has no subpackages. |
| Recommendation: bash-level integration test | — | Deferred to `todo.md`. |

### PR #62, head b034fed (round 3)

Critical: none. "Ready to merge? With fixes". Fixed in the next commit:

| Finding | Level | Action |
| --- | --- | --- |
| `no_secrets` `sk-` pattern matched prose like `task-auto-optimize-...` | Important | Token prefixes must start a word; tested with true and false positives. |
| No automated tests for the bash safety helpers | Important | Script is sourceable; `scripts/autooptimize/shell_test.go` drives `no_secrets`, `review_answer`/`relative_paths`, `revert_out_of_scope`, and `guard_snapshot`/`guard_check` (hook, git config, ignored file, run state, excludes) in temp repos under `make test`. |
| Bold sub-titles like "Summary of problem" ended the Critical section | Minor | Only the template's own section titles (prefix match) end a section. |
| `config.worktree` watched but not restored | Minor | Backed up and restored. Pre-existing ignored directories are not hashed file by file; the runner's checkout is fresh each run (documented in the code comment). |
| `gh label list` default page size | Minor | `--limit 100`. |

### PR #62, head f0d74fa (round 4)

Critical: none. "Ready to merge? With fixes". Fixed in the next commit:

| Finding | Level | Action |
| --- | --- | --- |
| On a secret hit, the flagged PR body/comment were still uploaded as artifacts | Important | `no_secrets` deletes the flagged files before `give_up`; tested. |
| `acpx` exit 0 after an API error counted as a finished turn, then an empty diff abandoned the candidate | Important | A turn counts only if it ends with `[done] end_turn` (exit 0 or 5); otherwise `give_up`. Tested with a stubbed `acpx`. |
| Verdict parser read acpx trace lines (`[done] end_turn` inside the last section) | Important | `reviewVerdict` skips trace lines; fixture `trace.md`. |
| Reused out dir kept other per-run files | Minor | Cleanup list covers every per-run file. |
| Pre-existing ignored dirs comment missing | Minor | Comment on `guard_state`. |
| `./`-prefixed tool paths bypassed exclusions; dry run without proxy | Minor | `cleanPath` (`path.Clean`), tested; dry run uses `AUTO_OPTIMIZE_PROXY` when set. |
