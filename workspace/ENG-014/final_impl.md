# ENG-014 Final Implementation

`new_impl.md` adopted. Change=-: no OpenSpec change; CI/infra tooling, scripts and docs only. No change to `cmd/`, `provider/`, `stats/`, `cursor/`, `e2e/` or the CLI contract.

Repository changes:
- `scripts/runner/job-hook.sh`, `scripts/runner/setup-pool.sh` (new)
- `.github/workflows/runner-smoke.yml` (pool check step, explicit `clean: true`, per-ref concurrency)
- `docs/self-hosted-runner.md` (pool, layout, reuse, hooks, cache caps, ephemeral comparison)
- `docs/plans/2026-06-13-engineering-roadmap.md` (ENG-014 row, Change=- mapping, change log)
- `workspace/ENG-014/*`

Machine changes (outside git, approved by the maintainer before running): `scripts/runner/setup-pool.sh` run once. See `verification.md`.

Security rules from ENG-011 unchanged: no new triggers, no credentials in repo or logs (registration token never printed), fork-PR approval policy untouched.
