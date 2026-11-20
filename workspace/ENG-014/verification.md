# ENG-014 Verification

Machine: Mac mini M4 (`Mac16,10`), 10 cores (4P + 6E), 32 GB RAM, 306 GB free. Runner 2.338.2.

## Pool install (approved by the maintainer before running)

`scripts/runner/setup-pool.sh --dry-run` was shown first, then `scripts/runner/setup-pool.sh` ran once:

- `~/actions-runner/_shared/{hooks,cache/go-mod,cache/go-build,cache/golangci-lint}` created; hooks installed.
- `codetok-mac-mini` (instance 1): managed `.env` block added, restarted while idle (ENG-011's smoke had finished).
- `codetok-mac-mini-2`, `codetok-mac-mini-3`: runner build copied from instance 1, registered with labels `codetok-local,ai`, LaunchAgents installed, `SessionCreate` removed, started.

Readback:

```text
$ gh api repos/Miss-you/codetok/actions/runners --jq '.runners[] | "\(.name) \(.status) busy=\(.busy)"'
codetok-mac-mini online busy=false
codetok-mac-mini-2 online busy=false
codetok-mac-mini-3 online busy=false
$ launchctl list | grep actions.runner      -> 3 LaunchAgents, exit status 0
$ PlistBuddy -c 'Print :SessionCreate' <instance 1 plist>   -> Does Not Exist
```

Labels on all three: `self-hosted, macOS, ARM64, codetok-local, ai`.

## Hook rehearsal (local, fake runner root)

`CODETOK_RUNNERS_ROOT=<tmp>`, a clone as `GITHUB_WORKSPACE` with a tracked edit, `bin/codetok`, an untracked `.env.local`, a nested git repo and an `http.https://github.com/.extraheader`; a read-only 3 MB module cache with a 1 MB cap.

- `job-hook.sh completed`: `git status --porcelain` empty, `bin/`, `.env.local`, nested repo gone, no `extraheader`, `.git` kept, module cache cleared, `prune.lock` released, exit 0.
- `GITHUB_WORKSPACE=/tmp` -> "outside the runner root, leaving it alone"; empty `GITHUB_WORKSPACE` -> "no workspace yet".

## Smoke runs on the pool

| Run | Job | Instance | Workspace | Shared caches | checkout | make build | make test | job total |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| [44762034309](https://github.com/Miss-you/codetok/actions/runs/44762034309) | single | codetok-mac-mini-2 | fresh clone | empty (cold) | 10 s | 24 s | 177 s | 236 s |
| [44762851544](https://github.com/Miss-you/codetok/actions/runs/44762851544) | slot 3 | codetok-mac-mini-2 | reused `.git` | warm | 2 s | 1 s | 0 s (results cached) | 31 s |
| same | slot 2 | codetok-mac-mini | reused `.git` (ENG-011 checkout) | warm deps | 3 s | 1 s | 234 s | 264 s |
| same | slot 1 | codetok-mac-mini-3 | fresh clone | warm deps | 4 s | 1 s | 233 s | 267 s |
| WARM_RUN | single | WARM_RUNNER | WARM_MODE | warm | WARM_CO | WARM_BUILD | WARM_TEST | WARM_JOB |

All green, including "Check runner pool setup" (`job hooks configured: yes`, `shared GOMODCACHE: yes`, `shared GOCACHE: yes`, `clean tree after checkout: yes`). Hook lines in the job logs, for example the cold run:

```text
Set up runner    [runner-hook:started] workspace emptied (no git checkout)
Complete runner  [runner-hook:completed] workspace reset (git clean -ffdx, reset --hard; .git kept for incremental fetch)
Complete runner  [runner-hook:completed] GOCACHE: 426MB <= 10240MB cap
Complete runner  [runner-hook:completed] GOMODCACHE: 371MB <= 4096MB cap
```

Concurrency: run 44762851544 (temporary 3-slot matrix) started all three jobs within 2 s of each other, one per instance.

## Local cold vs warm (same machine, outside the runner)

Fresh clone and empty `GOMODCACHE`/`GOCACHE` vs the same clone reused with warm caches (`HTTPS_PROXY` via the local proxy):

| Step | Cold | Warm |
| --- | --- | --- |
| checkout (full-history clone vs `git fetch` + `clean -ffdx` + `reset --hard`) | 4.9 s | 1.9 s |
| `make test`, nothing changed (test results cached) | 266.1 s | 0.2 s |
| `go test -race -cover -count=1 ./...` (build cache warm, tests rerun) | 266.1 s | 160.0 s |
| cache size after one run | 371 MB modules, 425 MB build | |

## Reading the numbers

- Checkout: reuse saves seconds (10 s -> 2-4 s on the runner). The repository is small; the gain grows with history.
- Shared caches: the dependency build (mostly `modernc.org/sqlite`) is shared across instances. `make build` on an instance that had never built before took 1 s instead of 24 s.
- Test results are cached only on the same instance: each instance compiles from its own absolute workspace path, so the test binaries (and so the test-result cache keys) differ across instances. Same instance, unchanged code: `make test` 0 s, whole job 31 s instead of 236 s.
- When tests must rerun, `make test` is dominated by `e2e` (about 150 s alone, 230 s with three race test runs in parallel). Each e2e test runs `go build` with a fresh empty `GOCACHE` (`e2e/e2e_test.go` `buildBinary`), about 6 s per test x 27 tests, so no runner cache can help it. Fixing that is test code, outside ENG-014: proposed as a follow-up.
- Three parallel `go test -race` runs slow each other down (177 s -> 233 s). This is the ceiling that keeps the pool at 3.

## Repository checks

MAKE_CHECKS
