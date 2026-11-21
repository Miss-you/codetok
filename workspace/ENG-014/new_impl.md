# ENG-014 New Implementation

1. Pool of 3 instances (`codetok`, `codetok-2`, `codetok-3`), each its own registration, `_work` and LaunchAgent, same labels. Instance 1 is the ENG-011 runner, kept in place (no re-registration). New instances are copies of instance 1's runner build (no download), registered with a one-time token from the local `gh` login.
2. Shared, runner-only Go caches in `~/actions-runner/_shared/cache` via a managed block in every instance's `.env`.
3. Job hooks (`ACTIONS_RUNNER_HOOK_JOB_STARTED/_COMPLETED`) from `scripts/runner/job-hook.sh`, copied to `_shared/hooks` so a checkout cannot change them: reset + `git clean -ffdx` before and after every job, scrub persisted checkout credentials, keep `.git` for incremental fetch; size-capped cache pruning only when no other instance is busy, under a mkdir lock; `_diag` logs older than 14 days.
4. `scripts/runner/setup-pool.sh`: idempotent installer with `--dry-run`, restarts only idle instances.
5. Smoke workflow: explicit `clean: true`, a "Check runner pool setup" step (hooks configured, shared caches, clean tree), per-ref concurrency so the pool can run smoke for different refs at once.

Alternatives considered: ephemeral runners (rejected, see `docs/self-hosted-runner.md` "Why not ephemeral runners"); per-instance caches (simpler pruning, but 3x cold builds and against the shared-cache goal); a separate launchd cleanup job (one more service; the completed hook already runs at the right moment).
