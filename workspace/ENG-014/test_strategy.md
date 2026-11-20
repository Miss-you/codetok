# ENG-014 Test Strategy

| Claim | Evidence |
| --- | --- |
| No product behavior change | `make test`, `make vet`, `make build` green |
| Hook cleans the tree and scrubs credentials | Local rehearsal on a fake runner root: tracked edit, `bin/`, untracked secret file, nested repo and `extraheader` all gone after `completed`; `.git` kept |
| Hook never touches paths outside the runner root | Rehearsal with `GITHUB_WORKSPACE=/tmp` and empty workspace: no-op |
| Cache cap works | Rehearsal with a 1 MB cap and a read-only 3 MB module cache: cleared; lock released |
| Setup script is safe | `--dry-run` reviewed by the maintainer before the real run |
| Pool online | `gh api .../actions/runners`: 3 runners online with `codetok-local`, `ai`; `launchctl list` shows 3 LaunchAgents; no `SessionCreate` in plists |
| Smoke green on the pool with reuse | Two smoke runs on the same instance: first cold (empty `_work`, empty shared caches), second warm; "Check runner pool setup" passes; step timings from the API |
| Concurrency | Smoke runs for two refs at once land on different instances |
| Before/after numbers | Local benchmark plus the runner step timings in `verification.md` |
