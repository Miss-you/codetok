# ENG-013 verification (2026-11-20, macOS arm64, go1.26.0, golangci-lint 2.11.4)

## Unit tests

- `go test -race ./scripts/autooptimize/` → ok (collect, select, gate, check, round, verdict, describe/marker, usage).
- `make fmt` (no diff), `make vet`, `make lint` (0 issues), `make build`, `make test` → see the PR description for the final run.

## Local dry run (this worktree)

`scripts/auto-optimize.sh --dry-run` → 0 open PRs; 12 candidates (10 deadcode in `cmd/`, `stats/`; 2 docs: `--token` missing from both READMEs); picked `32c95726c72c` deadcode `cmd/collect.go:12 collectUsageEvents`. Checkout unchanged.

## Local rehearsal 1 (`--no-pr`, throwaway worktree, real acpx/agy/ai-review, base `d706446`)

- First attempt failed: the agent tried a shell `grep`, the policy denied it, acpx exited 5. Fixed (prompt + exit-5 handling), see `final_impl.md`.
- Second attempt: agent (`claude-sonnet-5-5`) deleted `collectUsageEvents` and its single test (`cmd/collect.go` −4, `cmd/collect_test.go` −15). Scope check ok, `make test` and `make lint` passed with 0 fix rounds, commit `70c4fb7` on local `auto-optimize/20261120-32c95726c72c`.
- Review (`gemini-4-hydrogen`, `scripts/ai-review.sh`): Critical None, Important None, "Ready to merge? Yes" → gate passed. PR body generated; nothing pushed.

## Local rehearsal 2 (after the review fixes, commit `39dc00d`)

Same candidate, same 19-line deletion. Guard snapshots matched (no git/ignored/state changes), sandboxed `make test` (empty HOME, `GOPROXY=off`) and `make lint` passed with 0 fix rounds, review "Ready to merge? Yes" with Critical/Important None, commit `8560656` on a local branch, PR body and `AI review:` comment generated, nothing pushed. The generated comment contained absolute local `file://` paths from the reviewer's links; `relative_paths` now strips the checkout prefix (checked on that review: 0 absolute paths left).

## Gates on the final branch

`make fmt` (no diff), `make vet`, `make lint` (0 issues), `make build`, `make test` (all packages ok, `scripts/autooptimize` 84.1%).

## Not yet verified

- The workflow on the self-hosted runner (needs ENG-011 merged and the runner online) and PR mode (needs the `auto-optimize` label). See `todo.md`.
