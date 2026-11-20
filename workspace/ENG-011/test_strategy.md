# ENG-011 Test Strategy

What must be proven, and how:

| Claim | Evidence |
| --- | --- |
| No product behavior change | `make test` (and `make vet`, `make build`) green on the worktree |
| Workflow file is valid | YAML parses; GitHub accepts it (dispatch succeeds) |
| `agy` callable with a minimal env | Local rehearsal: the `agy -p` step script run under `env -i HOME USER PATH=/usr/bin:/bin:~/.local/bin:/opt/homebrew/bin AI_REVIEW_PROXY=...` |
| `ai-review.sh --dry-run` works with a minimal env | Same `env -i` rehearsal |
| Runner online with labels | `gh api repos/Miss-you/codetok/actions/runners` shows `codetok-mac-mini`, `online`, labels incl. `codetok-local`, `ai` |
| Runs as a launchd service as the login user | `./svc.sh status`, `launchctl list | grep actions.runner`, process owner |
| Real launchd environment works | `runner-smoke.yml` dispatched on the PR branch runs green on the runner; run URL recorded |
| No credentials in logs | Inspect smoke run log: env step prints tool paths only; agy step prints status + answer only |
| Fork PRs cannot reach the runner | Smoke workflow has no PR trigger; repository approval policy readback = `all_external_contributors`; documented guard for future PR-triggered jobs. Not tested with a real fork PR (no second account). |
| Auto-start after reboot | LaunchAgent `RunAtLoad` checked in plist. Real reboot left to the maintainer (needs a login because FileVault is on). |
