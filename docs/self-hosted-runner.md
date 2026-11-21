# Self-hosted macOS runner

codetok's regular CI (`.github/workflows/ci.yml`, `release.yml`) runs on GitHub-hosted
runners and stays there. A small pool of self-hosted runners on the maintainer's Mac exists only for
jobs that need this machine's local tools and login state: `acpx`, `agy`, `gh`, and the
local proxy. AI review (#54), post-merge archiving (#57) and automated optimization (#58)
build on it. Tracking issues: #56 (`ENG-011`, the runner) and #63 (`ENG-014`, the pool,
workspace reuse and per-job cleanup).

## Summary

| Item | Value |
| --- | --- |
| Scope | Repository runner, registered to `Miss-you/codetok` only |
| Instances | 3: `~/actions-runner/codetok` (`codetok-mac-mini`), `~/actions-runner/codetok-2` (`codetok-mac-mini-2`), `~/actions-runner/codetok-3` (`codetok-mac-mini-3`) |
| Labels | Same on every instance: `self-hosted`, `macOS`, `ARM64` (defaults) + `codetok-local`, `ai` |
| Service | One launchd LaunchAgent per instance, `actions.runner.Miss-you-codetok.<name>`, runs as the login user (not root) |
| Work dir | `<instance>/_work`, kept between jobs (incremental fetch), cleaned by job hooks |
| Shared | `~/actions-runner/_shared/hooks` (job hooks), `~/actions-runner/_shared/cache/{go-mod,go-build,golangci-lint}` |
| Setup | `scripts/runner/setup-pool.sh` (idempotent; `--dry-run` prints the commands) |
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

Every job on `[self-hosted, codetok-local]` also carries an owner check, so only runs the
repository owner triggered reach this runner:

```yaml
    if: >-
      github.repository == 'Miss-you/codetok' &&
      github.triggering_actor == github.repository_owner
```

(combine it with the `pull_request` guard above with `&&`). These jobs run on the
maintainer's Mac with the local `gh` / `agy` credentials. Today the owner is the only
collaborator; the check keeps the runner owner-only if collaborators are added later.

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

## Runner pool, workspace reuse and cleanup (ENG-014)

### Why 3 instances

The Mac is a Mac mini M4 (10 cores: 4 performance + 6 efficiency, 32 GB RAM). The jobs on
this runner are of two kinds:

- AI jobs (#57 archive, #58 optimize, AI review): mostly waiting on `agy` / the network,
  little CPU.
- `make test`: `go test -race` over all packages, CPU-bound for about 3 minutes when tests
  must run, dominated by the `e2e` package (about 150 s even with a warm build cache).

Three instances let the three workflow kinds (smoke, archive, optimize) run at the same
time. Measured: three parallel smoke jobs each took 233 s in `make test` against 177 s for
one alone, so the CPU is already shared at three; a fourth instance would only make
CPU-bound jobs slower and the Mac less usable for interactive work, while queuing
is rare at this repository's job volume. Each instance holds about 440 MB of runner build
plus its `_work`; memory stays far below 32 GB. Change the count with
`scripts/runner/setup-pool.sh --count N` (adding instances is safe; removing one is the
"Remove the runner" steps below for that directory).

### Layout

```text
~/actions-runner/
  codetok/      instance 1 (ENG-011), codetok-mac-mini
  codetok-2/    instance 2, codetok-mac-mini-2
  codetok-3/    instance 3, codetok-mac-mini-3
    .env        proxy + managed block (caches, hooks)   <- per instance, same content
    _work/codetok/codetok/   persistent checkout          <- per instance
  _shared/
    hooks/job-hook.sh, job-started.sh, job-completed.sh  <- copied from scripts/runner/
    cache/go-mod, cache/go-build, cache/golangci-lint     <- shared by all instances
```

`setup-pool.sh` writes this block into every instance's `.env` (between
`# >>> codetok runner pool` and `# <<<` markers), which the runner applies to its own
process and to every job:

```text
GOMODCACHE=~/actions-runner/_shared/cache/go-mod
GOCACHE=~/actions-runner/_shared/cache/go-build
GOLANGCI_LINT_CACHE=~/actions-runner/_shared/cache/golangci-lint
ACTIONS_RUNNER_HOOK_JOB_STARTED=~/actions-runner/_shared/hooks/job-started.sh
ACTIONS_RUNNER_HOOK_JOB_COMPLETED=~/actions-runner/_shared/hooks/job-completed.sh
```

(`.env` holds absolute paths; `~` is shorthand here.) The Go caches are separate from the
maintainer's own `~/go/pkg/mod` and `~/Library/Caches/go-build`, so pruning them never
touches interactive work. Both Go caches are safe for concurrent use by several `go`
processes.

What is shared and what is not: dependency builds (mostly `modernc.org/sqlite`) hit the
shared `GOCACHE` from any instance (`make build` 24 s cold, 1 s on an instance that never
built before). Test results hit only on the same instance, because each instance compiles
from its own absolute workspace path. Numbers: `workspace/ENG-014/verification.md`.

### Workspace reuse

The runner keeps `<instance>/_work/<repo>/<repo>` between jobs. When `.git` already
exists there, `actions/checkout` fetches only new objects into it instead of cloning. Use
these checkout settings in jobs on this runner:

```yaml
      - uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4
        with:
          fetch-depth: 0              # cheap on a reused .git; gives origin/main for merge-base
          clean: true                 # default; git clean -ffdx + reset --hard before checkout
          persist-credentials: false  # unless the job pushes
```

Keep the default checkout `path`; the hooks clean the default workspace only.

### Per-job cleanup (job hooks)

`scripts/runner/job-hook.sh` runs as the runner's job-started and job-completed hook. It
is copied to `_shared/hooks`, so a branch or PR cannot change what it executes; re-run
`setup-pool.sh` after changing it.

| Phase | What it does |
| --- | --- |
| started | Wait (up to 5 min) for a cache prune on another instance to finish. In `$GITHUB_WORKSPACE`: delete `.git/hooks` and code-running git config a previous job could have planted (`core.hooksPath`, `core.fsmonitor`, credential helpers, filter/diff drivers; git runs here with hooks and fsmonitor disabled), remove persisted checkout credentials (`http.*.extraheader`, `includeIf.*.path`), abort a half-done rebase/merge, `git reset --hard`, `git clean -ffdx` (also nested repos and ignored files). Keeps `.git`. Empties the workspace if it is not a git checkout. Catches leftovers when a previous job crashed. |
| completed | The same reset, so build outputs (`bin/`, coverage), AI scratch files and tokens do not sit on disk until the next job. Then bounded cache cleanup (below). |

The runner itself empties `_work/_temp` (`$RUNNER_TEMP`) for every job; put AI output
(such as `AI_REVIEW_OUT_DIR`) there.

The hook refuses to touch any path outside `~/actions-runner/`, prints only short status
lines (no paths, no environment), and always exits 0, so a cleanup problem is logged but
never fails a job.

### Bounded cache cleanup

At the end of each job the completed hook checks the shared caches against size caps and
clears a cache that is over its cap:

| Cache | Cap (MB) | Override in `.env` |
| --- | --- | --- |
| `GOCACHE` | 10240 | `CODETOK_RUNNER_GOCACHE_MAX_MB` |
| `GOMODCACHE` | 4096 | `CODETOK_RUNNER_GOMODCACHE_MAX_MB` |
| `GOLANGCI_LINT_CACHE` | 2048 | `CODETOK_RUNNER_LINTCACHE_MAX_MB` |
| `_diag/*.log` (all instances) | age 14 days | `CODETOK_RUNNER_DIAG_MAX_AGE_DAYS` |

`GOCACHE` also trims itself (Go drops entries unused for 5 days). A cache is cleared only
when no other instance is running a job (no other `Runner.Worker` process; if the check
cannot find even its own worker it assumes busy), only one instance prunes at a time
(`_shared/prune.lock`), and jobs starting meanwhile wait for the prune; otherwise the next
job retries. Only the fixed directories under `_shared/cache` are pruned, whatever a job
set `GOCACHE` to. Non-numeric caps fall back to the defaults. A cleared
cache makes the next job cold, not wrong. For codetok the caches stay well under the caps:
about 370 MB of modules and 430 MB of build cache.

### Why not ephemeral runners

Ephemeral runners (`config.sh --ephemeral`) take one job and unregister. On GitHub-hosted
or VM-based runners that gives a fresh machine per job. Here it would not:

- There is no VM layer on this Mac. An ephemeral runner still runs as the same user with
  the same `$HOME`, keychain, `agy`/`gh` login state and proxy, which is exactly what the
  AI jobs need. The isolation gain is a fresh `_work` directory, which the job hooks
  already provide.
- Every job would start with a full clone in a new work directory, so it would lose the
  per-instance test-result cache too (smoke job: 236 s cold vs 31 s on a reused instance
  with unchanged code, see `workspace/ENG-014/verification.md`).
- Something has to re-register a runner after each job: a long-running controller holding a
  credential that can mint registration tokens. That is more moving parts and a standing
  admin credential on the machine, for no security gain while fork PRs never reach this
  runner anyway.

Revisit this if the runner ever has to run untrusted code: the answer then is per-job VMs
(for example Tart on Apple silicon), not ephemeral runners on the host.

## Service management

Run from the instance directory (`~/actions-runner/codetok`, `codetok-2`, `codetok-3`):

```bash
./svc.sh status      # launchd state
./svc.sh stop
./svc.sh start
./svc.sh uninstall   # remove the LaunchAgent (keeps the registration)
```

Logs: `~/Library/Logs/actions.runner.Miss-you-codetok.<name>/` and `<instance>/_diag/`.
Hook output appears in the job log under "Set up runner" / "Complete runner".

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

- `_work` keeps checkouts between runs on purpose; the job hooks clean the tree. To start an
  instance from scratch (for example a corrupted `.git`), when it is idle:
  `./svc.sh stop && rm -rf _work/* && ./svc.sh start` in its directory.
- To reset the shared caches by hand, when no instance is busy:
  `chmod -R u+w ~/actions-runner/_shared/cache && rm -rf ~/actions-runner/_shared/cache/*`.
- After changing `scripts/runner/job-hook.sh`, re-run `scripts/runner/setup-pool.sh` to
  install it (it only restarts instances whose `.env` changed and that are idle).
- The runner updates itself automatically when GitHub releases a new version. Instances
  update independently.

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
| A job sees files from a previous job | The hooks are not configured on that instance. Check the job log for `[runner-hook:started]` lines and that its `.env` has the managed block; re-run `scripts/runner/setup-pool.sh`. |
| Smoke step "Check runner pool setup" fails | Same as above, or the instance was restarted before `setup-pool.sh` wrote `.env`. |
