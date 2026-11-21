# ENG-013 proposed implementation

Split the run into fixed steps and agent steps, and put every decision in code that `make test` covers.

- `scripts/autooptimize/` (Go, `package main`, stdlib only): subcommands `collect`, `select`, `describe`, `marker`, `gate`, `check`, `round`, `verdict`. Exit 0 = continue, 10 = end the run without a PR (not a failure), 1 = error, 2 = usage.
- `scripts/auto-optimize.sh`: glue that runs the tools (golangci-lint, deadcode, `go test -cover`), the change agent (`acpx` + `agy`, read/edit only, no terminal), `make test` / `make lint`, `scripts/ai-review.sh`, and `gh`. Modes `--dry-run`, `--no-pr`, default (PR).
- `.github/workflows/auto-optimize.yml`: `schedule` (opt-in via repo variable `AUTO_OPTIMIZE_SCHEDULE`) + `workflow_dispatch`, `runs-on: [self-hosted, codetok-local, ai]`, repository guard, no `pull_request` trigger, read-only `GITHUB_TOKEN`.
- `docs/auto-optimize.md`: trigger, constraints, credentials, settings, how to turn it off, merge-rate queries.

Alternatives considered:
- All bash with bats tests: bats is not installed and `make test` would not cover it.
- A Go program that also drives git/gh/acpx: more code to fake in tests for little gain; the glue has no decisions left in it.
- Review with the same agent: rejected by the issue (separate writer and reviewer). The writer is `claude-sonnet-5-5` via agy, the reviewer stays `gemini-4-hydrogen` via `scripts/ai-review.sh`.
