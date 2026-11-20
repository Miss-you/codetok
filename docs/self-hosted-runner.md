# Self-hosted macOS runner

codetok's regular CI (`.github/workflows/ci.yml`, `release.yml`) runs on GitHub-hosted
runners and stays there. One self-hosted runner, on the maintainer's Mac, exists only for
jobs that need this machine's local tools and login state: `acpx`, `agy`, `gh`, and the
local proxy. AI review (#54), post-merge archiving (#57) and automated optimization (#58)
build on it. Tracking issue: #56, task `ENG-011`.

## Summary

| Item | Value |
| --- | --- |
| Scope | Repository runner, registered to `Miss-you/codetok` only |
| Install dir | `~/actions-runner/codetok` (one directory per repository) |
| Runner name | `codetok-mac-mini` |
| Labels | `self-hosted`, `macOS`, `ARM64` (defaults) + `codetok-local`, `ai` |
| Service | launchd LaunchAgent `actions.runner.Miss-you-codetok.codetok-mac-mini`, runs as the login user (not root) |
| Work dir | `~/actions-runner/codetok/_work` |
| Smoke test | `.github/workflows/runner-smoke.yml` (manual `workflow_dispatch` only) |

## Conventions for workflows that use this runner

### `runs-on`

```yaml
runs-on: [self-hosted, codetok-local, ai]          # needs acpx / agy
runs-on: [self-hosted, codetok-local]              # needs only this Mac (gh, go)
```

Always include `codetok-local`, so a job never lands on some other self-hosted runner that
might be added later. Add `ai` for any job that calls `acpx` or `agy`. Regular build and
test jobs keep using `ubuntu-latest` / `macos-latest` / `windows-latest`.

### Triggers and the fork guard

The repository is public. A job on this runner executes repository code with the
maintainer's local credentials, so code from forks must never run here.

Allowed triggers:

- `push` restricted with `branches: [main]`
- `workflow_dispatch`, `schedule` (only users with write access can cause these)
- `workflow_run`, only with `if: github.event.workflow_run.event != 'pull_request'` or
  `github.event.workflow_run.head_repository.full_name == github.repository`
- `pull_request` from branches in this repository, with the guard below

Never use `pull_request_target` or `issue_comment` with a checkout of PR code on this
runner.

Every job on this runner that can be triggered by `pull_request` must carry the guard:

```yaml
jobs:
  example:
    if: >-
      github.event_name != 'pull_request' ||
      github.event.pull_request.head.repo.full_name == github.repository
    runs-on: [self-hosted, codetok-local, ai]
```

Add `github.repository == 'Miss-you/codetok'` as well when the job should do nothing in
forks of the repository.

Prefer `push` to `main` over `pull_request: closed` for "after merge" jobs: for every
`pull_request` event, `closed` included, GitHub runs the workflow file from the PR, while a
`push` to `main` always runs the file from `main` (`archive-records.yml` does this). Jobs on
this runner should also carry `github.triggering_actor == github.repository_owner` in their
`if:`, so they keep running only for the owner if collaborators are added later.

**The guard is hygiene, not the security boundary.** For `pull_request` events GitHub
runs the workflow files from the PR's merge commit, so a fork PR can delete the `if:` or
add its own job with `runs-on: [self-hosted, ...]`. GitHub advises against self-hosted
runners on public repositories for exactly this reason. The real boundary is:

1. Repository setting **Settings → Actions → General → Fork pull request workflows from
   outside collaborators** = **Require approval for all outside collaborators**
   (`approval_policy=all_external_contributors`). Every fork PR run waits for a
   maintainer.
2. The maintainer **never clicks "Approve and run" on a fork PR that changes
   `.github/workflows/`** (or anything a workflow executes, such as the `Makefile` or
   `scripts/`) without reading it: an approved run that targets this runner executes on
   this Mac with the local credentials.

Check the setting with:

```bash
gh api repos/Miss-you/codetok/actions/permissions/fork-pr-contributor-approval
```

### Environment and `PATH`

launchd starts the runner with a minimal environment. At configuration time `config.sh`
runs `env.sh`, which writes the `PATH` of the shell that ran `config.sh` to `~/actions-runner/codetok/.path`
and a few variables to `.env`; the runner applies them to every job. Do not rely on that
snapshot alone: the first step of every job pins the tool directories itself.

```yaml
      - name: Pin tool PATH
        run: |
          for dir in /opt/homebrew/sbin /opt/homebrew/bin "$HOME/go/bin" "$HOME/.local/bin"; do
            [ -d "$dir" ] && echo "$dir" >> "$GITHUB_PATH"
          done
```

| Tool | Location |
| --- | --- |
| `go`, `gh`, `acpx`, `node`, `python3` (Homebrew) | `/opt/homebrew/bin` |
| `agy` | `~/.local/bin` |
| `golangci-lint` | `~/go/bin` |

`HOME` is the login user's home, so `gh` and `agy` find their existing login state
(config files and the login keychain). Nothing else is needed.

### Proxy

`agy` ignores the macOS system proxy and reads only `HTTPS_PROXY` / `HTTP_PROXY`. Follow
the `scripts/ai-review.sh` convention:

```yaml
    env:
      AI_REVIEW_PROXY: ${{ vars.AI_REVIEW_PROXY || 'http://127.0.0.1:7890' }}
```

`scripts/ai-review.sh` reads `AI_REVIEW_PROXY` itself. A direct `agy` call exports
`HTTPS_PROXY=$AI_REVIEW_PROXY HTTP_PROXY=$AI_REVIEW_PROXY` and
`NO_PROXY=localhost,127.0.0.1,::1,.local` first (skip when the value is `none`), as
`runner-smoke.yml` does. The proxy client must be running on the Mac.

`vars.AI_REVIEW_PROXY` is a repository variable, which is not masked in logs: never put
proxy credentials (`user:pass@`) in it.

### Credentials and permissions

- No tokens in the repository, in workflow files, or in repository secrets for this
  runner. `acpx` / `agy` / `gh` use the login state already on the Mac.
- To write to the repository (push a branch, open a PR, comment), use the workflow's own
  `GITHUB_TOKEN`, and declare the smallest `permissions:` per job (the repository default
  is `read`).
- Use `actions/checkout` with `persist-credentials: false` unless the job pushes.
- Never `set -x` around commands that take tokens, and never print raw `gh auth token` or
  `agy` JSON payloads; print only the fields you need.
- Logs of a public repository are world-readable. The runner's work-directory path (and
  so the local user name) already shows up in `actions/checkout` output; do not add more:
  no home directory dumps, no full `PATH`, no `env`, and do not echo model output unchecked (it can inject
  `::workflow-commands::`).
- Pin third-party actions by commit SHA on this runner.

## Install (done once)

```bash
mkdir -p ~/actions-runner/codetok && cd ~/actions-runner/codetok
# Download actions-runner-osx-arm64-<version>.tar.gz from github.com/actions/runner releases,
# check its SHA-256 against the release notes, then:
tar xzf actions-runner-osx-arm64-<version>.tar.gz
token=$(gh api -X POST repos/Miss-you/codetok/actions/runners/registration-token --jq .token)
./config.sh --unattended --url https://github.com/Miss-you/codetok --token "$token" \
  --name codetok-mac-mini --labels codetok-local,ai --work _work
unset token
# The Mac reaches github.com only through the local proxy, and launchd does not pass the
# shell's proxy variables. The runner loads .env into its own and every job's environment.
cat >> .env <<'ENV'
HTTPS_PROXY=http://127.0.0.1:7890
HTTP_PROXY=http://127.0.0.1:7890
https_proxy=http://127.0.0.1:7890
http_proxy=http://127.0.0.1:7890
NO_PROXY=localhost,127.0.0.1,::1,.local
no_proxy=localhost,127.0.0.1,::1,.local
ENV
./svc.sh install
# svc.sh's plist sets SessionCreate=true, which gives the job its own security session
# without access to the login keychain, where agy keeps its login ("authentication
# required"; agy log: "Failed to load stored token from keyring ... exit status 36").
# Drop it so the runner runs in the user's login session, like a Terminal window.
/usr/libexec/PlistBuddy -c 'Delete :SessionCreate' \
  ~/Library/LaunchAgents/actions.runner.Miss-you-codetok.codetok-mac-mini.plist
./svc.sh start
```

Re-running `./svc.sh install` rewrites the plist; repeat the `PlistBuddy` line afterwards.

The registration token expires after one hour and is used only once; the runner then
keeps its own credentials in `.credentials` / `.runner` inside the install directory.
Never copy those files into a repository.

## Service management

Run from `~/actions-runner/codetok`:

```bash
./svc.sh status      # launchd state
./svc.sh stop
./svc.sh start
./svc.sh uninstall   # remove the LaunchAgent (keeps the registration)
```

Logs: `~/Library/Logs/actions.runner.Miss-you-codetok.codetok-mac-mini/` and
`~/actions-runner/codetok/_diag/`.

GitHub side:

```bash
gh api repos/Miss-you/codetok/actions/runners --jq '.runners[] | {name, status, busy, labels: [.labels[].name]}'
```

Remove the runner completely:

```bash
./svc.sh stop && ./svc.sh uninstall
token=$(gh api -X POST repos/Miss-you/codetok/actions/runners/remove-token --jq .token)
./config.sh remove --token "$token"; unset token
```

## Start on boot

`svc.sh` installs a **LaunchAgent** (`~/Library/LaunchAgents/actions.runner.*.plist`,
`RunAtLoad`), which launchd starts when the user session starts. Without automatic login
(always the case with FileVault), the runner comes online after a reboot once the
maintainer logs in. For an unattended reboot, use
`sudo fdesetup authrestart`, which unlocks the disk and logs in once.

A LaunchAgent, not a LaunchDaemon, is deliberate: the job needs the user's login keychain
and `agy` login state, which a daemon running before login cannot reach.

## Maintenance

- `_work` keeps checkouts between runs. Clean it now and then, when no job is running:
  `./svc.sh stop && rm -rf ~/actions-runner/codetok/_work/* && ./svc.sh start`.
- The runner updates itself automatically when GitHub releases a new version.

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `agy: command not found` / `acpx: command not found` | The job did not pin `PATH`. Add the "Pin tool PATH" step. Or re-run `./env.sh` from a login shell and `./svc.sh stop && ./svc.sh start` to refresh `.path`. |
| `actions/checkout` fails with `Empty reply from server` / `Failed to connect to github.com port 443` | The runner has no proxy. Add the proxy lines to `~/actions-runner/codetok/.env` (see Install) and `./svc.sh stop && ./svc.sh start`. |
| `agy` says the model is unknown, times out, or "User location is not supported" | No proxy or a proxy exit node in an unsupported region. Check `AI_REVIEW_PROXY` and that the proxy client is running (`nc -z 127.0.0.1 7890`). |
| Job stays "Waiting for a runner" | Runner offline (Mac asleep, user logged out, service stopped). Check `./svc.sh status` and the Runners page. Prevent sleep with `caffeinate` or Energy settings if the jobs are scheduled. |
| `agy` fails with `authentication required` only under the runner | The LaunchAgent still has `SessionCreate=true` (e.g. after `./svc.sh install`). Delete the key with `PlistBuddy` (see Install) and `./svc.sh stop && ./svc.sh start`. |
| `agy` asks to log in | The login state expired. Log in again interactively as the same user; the service picks it up. |
| A fork PR shows "approval required" | Expected. Do not approve a fork PR run that would target this runner; the fork guard skips such jobs anyway. |
