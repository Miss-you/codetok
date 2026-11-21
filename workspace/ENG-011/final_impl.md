# ENG-011 Final Implementation

`new_impl.md` was adopted unchanged (the task is small and bounded; no `final_impl_v1.md` iteration needed beyond the review in `review.md`).

Change=-: no OpenSpec change. This is CI/infra tooling plus docs. It does not touch `cmd/`, `provider/`, `stats/`, `cursor/` or `e2e/`, the CLI contract, or token-counting semantics.

Repository changes:
- `.github/workflows/runner-smoke.yml` (new, `workflow_dispatch` only)
- `docs/self-hosted-runner.md` (new)
- `docs/plans/2026-06-13-engineering-roadmap.md` (ENG-011 row + Change Log)
- `workspace/ENG-011/*`

Machine / repository changes (outside git, approved by the maintainer before running; see `verification.md`):
- Runner installed in `~/actions-runner/codetok`, registered as `codetok-mac-mini`, labels `codetok-local,ai`, LaunchAgent installed and started.
- Fork-PR approval policy: `first_time_contributors` → `all_external_contributors`.

Downstream contract for #57 / #58: see "Conventions for workflows that use this runner" in `docs/self-hosted-runner.md`.
