# ENG-011 Verification

Date: 2026-11-20 CST, worktree branch `claude/eng-011-self-hosted-runner-e8340a`.

## Repository gates

```bash
make test
# go test -race -cover ./...
# ok  github.com/miss-you/codetok/cmd, cursor, e2e (312s), provider, provider/{claude,codex,cursor,kimi}, stats
make vet
# go vet ./...   (clean)
```

No Go code changed; gates confirm no regression.

## Runner registration (machine + repo, approved by the maintainer before running)

- Package `actions-runner-osx-arm64-2.338.2.tar.gz`, SHA-256 checked against the release notes (`25a065af…a388`: OK), extracted to `~/actions-runner/codetok`.
- `config.sh --unattended --name codetok-mac-mini --labels codetok-local,ai --work _work` (token from `gh api …/registration-token`, held in a shell variable, never printed).
- `svc.sh install && svc.sh start` → LaunchAgent `actions.runner.Miss-you-codetok.codetok-mac-mini`, `RunAtLoad=true`, process `Runner.Listener run --startuptype service` owned by the login user (not root).

```bash
gh api repos/Miss-you/codetok/actions/runners --jq '.runners[] | {name, os, status, busy, labels: [.labels[].name]}'
# {"busy":false,"labels":["self-hosted","macOS","ARM64","codetok-local","ai"],"name":"codetok-mac-mini","os":"macOS","status":"online"}
```

Repository setting:

```bash
gh api -X PUT repos/Miss-you/codetok/actions/permissions/fork-pr-contributor-approval -f approval_policy=all_external_contributors
gh api repos/Miss-you/codetok/actions/permissions/fork-pr-contributor-approval
# {"approval_policy":"all_external_contributors"}   (was first_time_contributors)
```

## Smoke workflow on the runner

`workflow_dispatch` needs the workflow on the default branch, so pre-merge verification used a temporary commit adding `push: branches: [claude/eng-011-self-hosted-runner-e8340a]`, reverted in the next commit (the merged file is `workflow_dispatch` only).

Run: https://github.com/Miss-you/codetok/actions/runs/44733444119 (attempt 3, job https://github.com/Miss-you/codetok/actions/runs/44733444119/job/136285653497): **success**. Every step passed: checkout, Pin tool PATH, Check tools (all `found`), `make build`, `make test`, `ai-review.sh --dry-run`, `agy -p minimal call` (`agy call: PASS`).

Two launchd-environment problems were found by attempts 1 and 2 and fixed on the runner (both now in `docs/self-hosted-runner.md`):

1. Attempt 1: `actions/checkout` failed with `Failed to connect to github.com port 443`. This Mac reaches GitHub only through the local proxy, and launchd does not pass the shell's proxy variables. Fix: proxy variables (`HTTPS_PROXY`/`HTTP_PROXY`, lowercase variants, `NO_PROXY`) added to `~/actions-runner/codetok/.env`, service restarted.
2. Attempt 2: everything passed except `agy`: `Error: authentication required`. The agy log showed `Failed to load stored token from keyring … exit status 36`: svc.sh's plist sets `SessionCreate=true`, which isolates the job from the login keychain. Fix: `PlistBuddy -c 'Delete :SessionCreate'` on the LaunchAgent plist, service restarted.

Credential scan of the attempt-3 log (`gh run view --log`): searched for `ghs_|gho_|ghp_|github_pat|bearer|password|ya29.|refresh_token|access_token`. The only hit is the literal `$GITHUB_PATH` in the PATH step, a false positive. The checkout auth header appears only as GitHub's masked `AUTHORIZATION: basic ***`. The agy step prints only `agy call: PASS`.

## Local rehearsal (before the runner existed)

- `env -i HOME USER PATH=/usr/bin:/bin:~/.local/bin AI_REVIEW_PROXY=http://127.0.0.1:7890 bash -e <agy step>` → `agy call: PASS`.
- `env -i … scripts/ai-review.sh --dry-run HEAD~1 HEAD` → prompt written, acpx command printed.

## Fork PRs

Not tested with a real fork (that needs a second account). Covered by configuration: the smoke workflow has no PR trigger (`workflow_dispatch` needs write access), the job requires `github.repository == 'Miss-you/codetok'`, the fork-PR approval policy is `all_external_contributors`, and the doc defines the guard plus the rule "never approve fork runs that change workflows".

## Not verified here

- Reboot → auto-online: `RunAtLoad=true` is confirmed in the plist. A real reboot needs the maintainer to log in (FileVault), so it is left as a maintainer check in `todo.md`.
