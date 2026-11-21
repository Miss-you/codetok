# ENG-014 Review

An independent review subagent read `scripts/runner/job-hook.sh`, `scripts/runner/setup-pool.sh` and `runner-smoke.yml`. It found nothing high-severity: no token leak and no way to delete files outside `~/actions-runner`.

Fixed:
- Cache prune vs a job starting on another instance: the started hook now waits for `prune.lock`.
- Busy check failed open when nothing matched: it now assumes busy when even its own worker is not found. A real job then showed that macOS `pgrep` skips its own ancestors, so the count now comes from `ps`.
- Kept `.git` could carry code into the next job (`.git/hooks`, `core.fsmonitor`, `core.hooksPath`, credential helpers, filter/diff drivers): these are removed, and the hook runs git with hooks and fsmonitor disabled.
- `setup-pool.sh` lost a restart it skipped for a busy instance: it now leaves a `.pool-restart-pending` marker. `PlistBuddy Delete` no longer aborts when the key is absent.
- Low: prune paths are pinned to `_shared/cache` (a job can change `GOCACHE` through `GITHUB_ENV`); the workspace must be under a `_work` directory; non-numeric caps fall back to the defaults; `rm`/`mkdir` errors are not printed (they would show paths) and are reported as "clearing failed".
- The temporary push trigger was removed in the last commit.

Accepted:
- The registration token is visible in the local `ps` output while `config.sh` runs. It is single-use, expires in an hour, and this is a single-user Mac. It is never logged.
- `setup-pool.sh` takes the proxy lines from instance 1 for every instance. This is intended: all instances share one proxy.
- A start marker without an end marker in `.env` would drop the lines after it. Only the script writes these markers.
