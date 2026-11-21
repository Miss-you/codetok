# ENG-011 New Implementation

Goal: give later automation (#57 archive, #58 optimize) a trusted place to run that has this Mac's tools and login state, without changing the hosted CI and without exposing the Mac to fork code.

1. Register one repository-level runner, `codetok-mac-mini`, in `~/actions-runner/codetok`, labels `codetok-local` and `ai` on top of `self-hosted, macOS, ARM64`. Install with `svc.sh` (launchd LaunchAgent, runs as the login user).
2. Security: (a) runner jobs only run on trusted triggers, with the fork guard `github.event.pull_request.head.repo.full_name == github.repository` on any `pull_request` job; (b) repository fork-PR approval policy raised to `all_external_contributors`; (c) no secrets; `GITHUB_TOKEN` with minimal `permissions`.
3. Smoke workflow `.github/workflows/runner-smoke.yml`, `workflow_dispatch` only: pin PATH → env report → `make build` → `make test` → `scripts/ai-review.sh --dry-run` → one `agy -p` JSON call through `AI_REVIEW_PROXY`, printing only status + answer.
4. `docs/self-hosted-runner.md`: install, labels, service management, security boundary, conventions for downstream workflows (`runs-on`, guard, PATH, proxy, credentials), troubleshooting.

Alternatives rejected:
- LaunchDaemon (starts before login): cannot read the login keychain / `agy` login state, and with FileVault the Mac waits at the unlock screen after reboot anyway.
- Moving the existing hosted CI here: out of scope per #56.
- Relying only on `.path` from `env.sh`: a snapshot of the shell at config time; jobs pin PATH themselves so they keep working after tool moves.
