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
