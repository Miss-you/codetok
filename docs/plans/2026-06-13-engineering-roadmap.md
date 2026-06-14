# Engineering Roadmap Task Board

## Source Design

- Source: this file (no separate `*-design.md` — the roadmap is a backlog of small, independent engineering tasks rather than one feature design)
- Initialized: 2026-06-13 CST
- Approval signal: requested by the maintainer after manually reviewing repo state on `origin/main` (commit reachable from `main` as of 2026-06-13).
- Scope: open-source maintainability and AI-coding ergonomics for `codetok`. Each task is a small, claimable unit that does **not** change the daily/session token-counting semantics described in [AGENTS.md](file:///Users/apple/Documents/Github/codetok/AGENTS.md).
- Non-goals: do not change token-counting formulas, do not add remote provider APIs to `daily` / `session` / `cursor activity`, do not rename existing CLI commands, do not break existing JSON fields. Anything that does belongs in its own `*-design.md` and goes through [compatibility-first-planning](file:///Users/apple/Documents/Github/codetok/.claude/skills/compatibility-first-planning/SKILL.md).

## Principles (Acceptance Lens)

These are the standards used to evaluate every task in the table below. A task only earns a slot here if it materially improves at least one of these.

1. **Local-only contract is sacred.** `daily`, `session`, `cursor activity` read local files only. Network access lives behind explicit `cursor login/status/sync` commands. See [AGENTS.md](file:///Users/apple/Documents/Github/codetok/AGENTS.md) "Statistics scope".
2. **Snapshot honesty.** Reports over active local logs are point-in-time. Errors that mean "I could not read this file" must not silently become "this user has zero tokens".
3. **AI-agent-friendly CLI.** Help text, error messages, and JSON shape must be discoverable and machine-parseable. See [2026-04-15-ai-agent-friendly-cli-ux-improvements.md](file:///Users/apple/Documents/Github/codetok/docs/plans/2026-04-15-ai-agent-friendly-cli-ux-improvements.md) for the standard already accepted.
4. **Compatibility by default.** Add aliases instead of renaming. Keep JSON fields stable unless an explicit schema migration is approved.
5. **One verification gate per task.** Each task names the exact `make` target or focused test that proves it.
6. **Boundaries follow directories.** `cmd/` is shell, `provider/` is parsing, `stats/` is aggregation, `cursor/` is local + sync, `e2e/` is contract. Tasks should not cross boundaries unless explicitly stated.
7. **Standardized delivery flow is mandatory.** Every non-trivial task follows the Stage Loop defined in [delivering-go-task-end-to-end](file:///Users/apple/Documents/Github/codetok/.claude/skills/delivering-go-task-end-to-end/SKILL.md). Trivial tasks (single-file, no behavior change) may skip workspace artifacts but must still pass the named verification gate.

## Definition of Ready (DoR)

A task can move from `todo` to `claimed` only when **all** of the following are true:

1. All hard dependencies in `Depends On` are `done`.
2. The task title describes one closed behavior, not a directory rename.
3. `Done When` is checkable without re-reading this file's history.
4. The task either maps to an existing OpenSpec change in [openspec/changes/](file:///Users/apple/Documents/Github/codetok/openspec/changes), or `Change=-` is set and `Notes` explains why no OpenSpec delta is needed (typically: tooling, CI, docs, or test-only).
5. An owner is willing to follow the Stage Loop end-to-end, not just push a code change.

## Definition of Done (DoD)

A task moves to `done` only when **all** of the following are recorded:

1. The Stage Loop has reached `review` with no outstanding must-fix findings.
2. Named verification gate in `Notes` has passed fresh on the worktree (record date + command).
3. Where applicable, `workspace/<task-id>/` contains `original_impl.md`, `new_impl.md`, `final_impl_v1.md`, `final_impl.md`, `test_strategy.md`. Skipped artifacts are justified in [delivering-go-task-end-to-end](file:///Users/apple/Documents/Github/codetok/.claude/skills/delivering-go-task-end-to-end/SKILL.md) terms.
4. If the task changes user-visible behavior, [README.md](file:///Users/apple/Documents/Github/codetok/README.md) and [README_zh.md](file:///Users/apple/Documents/Github/codetok/README_zh.md) are updated in the same change.
5. If the task touches a contract surface (CLI flags, JSON fields, file roots), an `e2e/` test or focused CLI smoke test asserts the contract.
6. `Change Log` below has a one-line entry on every state transition.

## Status Legend

- `todo`: not claimed and not started.
- `claimed`: owner has claimed the task, recorded a workspace, and is preparing work.
- `research`: owner is gathering local evidence before implementation.
- `spec`: owner is writing or refining tests and acceptance details.
- `implementing`: owner is changing code or docs.
- `verifying`: owner is running validation and fixing failures.
- `review`: task is ready for review or awaiting review feedback.
- `blocked`: task cannot proceed; `Notes` must include `resume_to=<state>` and the blocker.
- `done`: implementation and listed `Done When` checks are complete with evidence.

## Dependency Rules

- Only claim tasks in `todo` when every hard dependency in `Depends On` is `done`.
- `ENG-001` is foundational because it changes the contract of `provider.ParseParallel`; provider tasks that surface read failures depend on it.
- `ENG-002` (Windows CI parity) is independent and can be claimed in parallel.
- `ENG-006` (`codetok sources` command) depends on `ENG-001` for honest "this root failed to read" reporting.
- `ENG-009` (contributor docs) depends on `ENG-007` (issue/PR templates) so links resolve.
- `ENG-010` (release rehearsal SOP) is the final gate — it depends on Windows CI parity (`ENG-002`) and source-scoped errors (`ENG-001`) so the release pipeline actually validates what we ship.
- If a task reveals the split is wrong, update this board and append a `Change Log` entry **before** continuing.

## Current Repo Inventory (as of 2026-06-13)

This snapshot is what the table below was built against. Refresh it when re-deriving tasks.

- **Commands present** ([cmd/](file:///Users/apple/Documents/Github/codetok/cmd)): `daily`, `session`, `cursor`, `collect` (helper), `version`. No `sources` command yet.
- **Providers present** ([provider/](file:///Users/apple/Documents/Github/codetok/provider)): `claude`, `codex`, `cursor`, `kimi`. Registry in [registry.go](file:///Users/apple/Documents/Github/codetok/provider/registry.go).
- **Event-based aggregation**: shipped (see [stats/events.go](file:///Users/apple/Documents/Github/codetok/stats/events.go) and EBTA-001..010 in [2026-04-16-event-based-token-aggregation-task.md](file:///Users/apple/Documents/Github/codetok/docs/plans/2026-04-16-event-based-token-aggregation-task.md)). Do **not** redo this work.
- **AI-agent CLI UX**: standard accepted (10 items, see [2026-04-15 plan](file:///Users/apple/Documents/Github/codetok/docs/plans/2026-04-15-ai-agent-friendly-cli-ux-improvements.md)). Some items (e.g. `sources` command, provider validation) appear here as concrete tasks where the implementation is still missing or partial.
- **OpenSpec changes active** ([openspec/changes/](file:///Users/apple/Documents/Github/codetok/openspec/changes)): 11 changes, including `provider-output-rename`, `report-reasoning-column`, `token-usage-output-split`, `codex-reasoning-parsing`. These are **already owned** — this roadmap does not duplicate them.
- **CI** ([.github/workflows/ci.yml](file:///Users/apple/Documents/Github/codetok/.github/workflows/ci.yml)): lint + test on `ubuntu-latest` only; build matrix is `{linux, darwin} x {amd64, arm64}`. No Windows runner.
- **Release** ([.goreleaser.yml](file:///Users/apple/Documents/Github/codetok/.goreleaser.yml)): ships `windows` binaries. Mismatch with CI matrix is the basis for `ENG-002`.
- **Silent skip on read failure**: confirmed in [provider/parallel.go](file:///Users/apple/Documents/Github/codetok/provider/parallel.go#L38-L51) (`return // skip failed items`) and [cmd/collect.go](file:///Users/apple/Documents/Github/codetok/cmd/collect.go#L31-L35) (`os.IsNotExist` continue). Basis for `ENG-001`.
- **Contributor surface**: no [.github/ISSUE_TEMPLATE/](file:///Users/apple/Documents/Github/codetok/.github), no [CONTRIBUTING.md](file:///Users/apple/Documents/Github/codetok/CONTRIBUTING.md), no PR template. Basis for `ENG-007`, `ENG-009`.
- **Standardized dev flow**: present in [.claude/skills/](file:///Users/apple/Documents/Github/codetok/.claude/skills) and [.codex/skills/](file:///Users/apple/Documents/Github/codetok/.codex/skills) (`compatibility-first-planning`, `deriving-task-board-from-design`, `delivering-go-task-end-to-end`, `claiming-and-delivering-work`, `writing-commit`, `writing-ginkgo-bdd-tests`, `monitoring-pr-ai-reviews`, `scanning-project-secrets`). Reference these in `Notes`, do not re-invent them.

## Task Table

| ID | Title | Goal | Depends On | Parallel | Status | Owner | Claimed At | Workspace | Change | Done When | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| ENG-001 | Surface provider read failures instead of silent skip | Replace blanket `return // skip failed items` in [provider/parallel.go](file:///Users/apple/Documents/Github/codetok/provider/parallel.go) and `os.IsNotExist`-only continues in [cmd/collect.go](file:///Users/apple/Documents/Github/codetok/cmd/collect.go) with a structured "per-root error" channel so partial failures (corrupt JSONL, permission denied, transient ENOENT mid-scan) are reported on stderr and reflected in a non-zero exit option, while empty roots still succeed. | none | No, foundation | todo | - | - | `workspace/ENG-001/` | propose new openspec change `provider-read-error-reporting` | `provider.ParseParallel` and `cmd/collect.go` expose read errors via a typed list; new focused tests cover (a) corrupt JSONL surfaced, (b) ENOENT root still succeeds, (c) PermissionDenied surfaced; `go test ./provider ./cmd -run 'Test(ParseParallel\|CollectSessions\|CollectUsageEvents)'` passes. | First verification gate: `go test ./provider ./cmd -run 'TestParseParallel\|TestCollectUsageEvents'`. Must keep current behavior that a single bad file does **not** crash the whole run; the change is _visibility_, not strictness. |
| ENG-002 | Windows CI parity with release | Add `windows-latest` runner to the `test` job and `windows/{amd64,arm64}` to `build_matrix` in [.github/workflows/ci.yml](file:///Users/apple/Documents/Github/codetok/.github/workflows/ci.yml) so CI proves what [.goreleaser.yml](file:///Users/apple/Documents/Github/codetok/.goreleaser.yml) ships. | none | Yes, infra | claimed | Trae | 2026-06-15 CST | `workspace/ENG-002/` | - | CI workflow runs `go test ./...` on `windows-latest` and produces `codetok-windows-{amd64,arm64}.exe` build artifacts; any path-handling/test bug uncovered is fixed in same change or filed as a follow-up task. | First verification gate: green CI run on a PR branch covering all 3 OS. `Change=-` because this is CI tooling, no product behavior changes. Watch for path separator and `\r\n` test failures. |
| ENG-003 | Adopt and document `make ci` local gate | Add a `make ci` target that runs `make fmt`, `make vet`, `make test`, `make build`, and `make lint` (lint is optional via `golangci-lint version` check) so the contract in [delivering-go-task-end-to-end](file:///Users/apple/Documents/Github/codetok/.claude/skills/delivering-go-task-end-to-end/SKILL.md) has a one-liner. | none | Yes, tooling | todo | - | - | `workspace/ENG-003/` | - | `make ci` exists in [Makefile](file:///Users/apple/Documents/Github/codetok/Makefile), runs all five gates, returns non-zero if any fails, skips `make lint` cleanly when `golangci-lint` is not installed; [AGENTS.md](file:///Users/apple/Documents/Github/codetok/AGENTS.md) "Build, Test, Validate" section lists it. | First verification gate: `make ci` on a clean checkout, then again with `PATH=` to confirm lint-skip path. `Change=-` because it is pure tooling. |
| ENG-004 | Provider validation error rewrites stale README claim | Apply item #6 from [2026-04-15 plan](file:///Users/apple/Documents/Github/codetok/docs/plans/2026-04-15-ai-agent-friendly-cli-ux-improvements.md): validate `--provider` against `provider.Registry()` before collection, return allowed list on typo, and update README examples that currently imply typos return empty data. | ENG-001 | Yes, CLI ergonomics | todo | - | - | `workspace/ENG-004/` | reuse plan from `2026-04-15-ai-agent-friendly-cli-ux-improvements.md` (no new openspec change needed if it stays within accepted standard) | `codetok daily --provider bogus` and `codetok session --provider bogus` exit non-zero, print the allowed list, and don't dump full usage; a valid provider with empty data still exits zero; `go test ./cmd -run TestProviderValidation` passes. | Depends on ENG-001 so that "valid provider, unreadable data" is no longer indistinguishable from "invalid provider". |
| ENG-005 | Token-field legend in help and READMEs | Apply items #2 and #4 from [2026-04-15 plan](file:///Users/apple/Documents/Github/codetok/docs/plans/2026-04-15-ai-agent-friendly-cli-ux-improvements.md): rewrite date wording to `YYYY-MM-DD, e.g. 2026-04-15` and add a compact token-field legend (`input_other`, `input_cache_read`, `input_cache_creation`, `output`, `input_total`, `total`) to `daily --help`, `session --help`, [README.md](file:///Users/apple/Documents/Github/codetok/README.md), [README_zh.md](file:///Users/apple/Documents/Github/codetok/README_zh.md). | none | Yes, docs | todo | - | - | `workspace/ENG-005/` | - | `codetok daily --help` and `codetok session --help` show date wording in `YYYY-MM-DD` form with example, and include the six-field legend; READMEs mirror it; help-text snapshot test (or focused string test) asserts the legend tokens. | First verification gate: `go test ./cmd -run TestHelpText` plus manual `./bin/codetok daily --help`. `Change=-` because it is help-text only. |
| ENG-006 | `codetok sources` local inventory command | Apply item #7 from [2026-04-15 plan](file:///Users/apple/Documents/Github/codetok/docs/plans/2026-04-15-ai-agent-friendly-cli-ux-improvements.md): create [cmd/sources.go](file:///Users/apple/Documents/Github/codetok/cmd/sources.go) that reports each provider's resolved local roots, whether they exist, and discovered file/session counts, with no network access. | ENG-001, ENG-004 | No, command surface | todo | - | - | `workspace/ENG-006/` | new openspec change `local-source-inventory-command` | `codetok sources` and `codetok sources --provider codex` print resolved roots + existence + counts + per-root read errors (using ENG-001 surface); missing-directory case still exits zero with explicit "(not found)"; e2e test `TestSourcesCommand` passes. | Depends on ENG-001 because surfaced read errors are first-class output here. |
| ENG-007 | GitHub issue and PR templates | Create [.github/ISSUE_TEMPLATE/bug.yml](file:///Users/apple/Documents/Github/codetok/.github/ISSUE_TEMPLATE/bug.yml), [feature.yml](file:///Users/apple/Documents/Github/codetok/.github/ISSUE_TEMPLATE/feature.yml), and [.github/pull_request_template.md](file:///Users/apple/Documents/Github/codetok/.github/pull_request_template.md). Issue templates must ask for codetok version, OS, provider involved, and a redacted JSONL snippet. PR template must reference the task ID and DoD checklist from this file. | none | Yes, governance | todo | - | - | `workspace/ENG-007/` | - | New files render correctly on GitHub (visual check on a draft PR); PR template enforces "linked task ID" and "checklist of DoD items 1-6 above"; no template asks for secrets. | First verification gate: open a throwaway draft issue and PR; record screenshot or URL in workspace. `Change=-` (governance only). |
| ENG-008 | Repo-wide secret scan baseline | Run [scanning-project-secrets](file:///Users/apple/Documents/Github/codetok/.codex/skills/scanning-project-secrets/SKILL.md) over the entire tracked file set, record `workspace/ENG-008/secret-scan-tasks.md`, and produce a PASS/FAIL baseline. Any confirmed secret moves to a blocker task immediately. | none | Yes, security | todo | - | - | `workspace/ENG-008/` | - | `workspace/ENG-008/secret-scan-tasks.md` exists with grouped checkboxes covering 100% of `git ls-files`, all 7 scan patterns applied, every group returns explicit PASS, and the final summary lists 0 confirmed secrets (or the corresponding remediation task IDs). | First verification gate: file-count sanity check (`git ls-files | wc -l` == sum of group counts). `Change=-` (security audit). |
| ENG-009 | CONTRIBUTING and developer onboarding doc | Create [CONTRIBUTING.md](file:///Users/apple/Documents/Github/codetok/CONTRIBUTING.md) that points to [AGENTS.md](file:///Users/apple/Documents/Github/codetok/AGENTS.md) (CLI contract), [.claude/skills/](file:///Users/apple/Documents/Github/codetok/.claude/skills) (delivery flow), this roadmap file, and `make ci`. Must explain how to claim a task from the table above, how to create `workspace/<task-id>/`, and where local session fixtures live ([e2e/testdata/](file:///Users/apple/Documents/Github/codetok/e2e/testdata)). | ENG-003, ENG-007 | No, depends on tooling and templates | todo | - | - | `workspace/ENG-009/` | - | `CONTRIBUTING.md` exists; sections cover "Project contract", "Standardized delivery flow", "Claiming a task", "Local validation (`make ci`)", "Opening a PR" with valid links; `README.md` and `README_zh.md` link to it from "Contributing". | First verification gate: link-checker (`go install github.com/raviqqe/muffet@latest && muffet --local CONTRIBUTING.md`) or manual review. `Change=-` (docs only). |
| ENG-010 | Release rehearsal SOP and dry-run script | Extend [skills/codetok-release-sop/SKILL.md](file:///Users/apple/Documents/Github/codetok/skills/codetok-release-sop/SKILL.md) with a dry-run rehearsal step that runs `goreleaser release --snapshot --clean` locally, validates the produced [npm/](file:///Users/apple/Documents/Github/codetok/npm) layout via [npm/scripts/install.mjs](file:///Users/apple/Documents/Github/codetok/npm/scripts/install.mjs), and asserts that `windows`, `linux`, `darwin` archives all exist before any tag is pushed. | ENG-002 | No, final gate | todo | - | - | `workspace/ENG-010/` | - | Release SOP has an explicit "Step 0: dry-run rehearsal"; rehearsal completes on a clean checkout, all 6 archives (3 OS x 2 arch) are produced, and `npm/scripts/install.mjs` resolves a fake "current platform" binary from each; SOP names exactly one command-line transcript file to attach to the release issue. | Depends on ENG-002 so we don't dry-run-release Windows binaries that CI never tested. First verification gate: `goreleaser release --snapshot --clean` on a clean macOS checkout, then inspect `dist/`. |

## Claiming Rules

- Before working, read this board, the linked principles, and the relevant skill in [.claude/skills/](file:///Users/apple/Documents/Github/codetok/.claude/skills).
- Claim exactly one `todo` task whose hard dependencies are all `done`.
- Update the task row **first**: set `Status` to `claimed`, set `Owner`, set `Claimed At`, set `Workspace` to `workspace/<task-id>/`.
- Create the workspace directory **after** updating the row.
- Append a `Change Log` entry describing the claim.
- Do not edit another active task's workspace or revert unrelated repository changes.
- If implementation reveals the task split is wrong, update this board and append a `Change Log` entry **before** continuing.
- A task can move to `done` only when its `Done When` checks are satisfied **and** the relevant validation output is recorded in `Notes`, workspace notes, or the commit message — per DoD above.

## OpenSpec Change Mapping

This roadmap intentionally does **not** duplicate the OpenSpec changes already active in [openspec/changes/](file:///Users/apple/Documents/Github/codetok/openspec/changes). Owners of those changes (provider-output-rename, report-reasoning-column, token-usage-output-split, codex-reasoning-parsing, etc.) continue on their own task boards under [docs/plans/](file:///Users/apple/Documents/Github/codetok/docs/plans).

Mapping summary:

| Task | OpenSpec change |
| --- | --- |
| ENG-001 | New: `provider-read-error-reporting` (to be created with `openspec-ff-change` during the spec stage of ENG-001) |
| ENG-002 | none (`Change=-`, CI tooling) |
| ENG-003 | none (`Change=-`, Makefile only) |
| ENG-004 | reuse standard accepted in `docs/plans/2026-04-15-ai-agent-friendly-cli-ux-improvements.md` |
| ENG-005 | none (`Change=-`, help text + docs only) |
| ENG-006 | New: `local-source-inventory-command` |
| ENG-007 | none (`Change=-`, governance only) |
| ENG-008 | none (`Change=-`, security audit) |
| ENG-009 | none (`Change=-`, docs only) |
| ENG-010 | none (`Change=-`, SOP only, but must reference ENG-002 evidence) |

If an owner of ENG-001 or ENG-006 decides the change is too small for OpenSpec, they must document the decision in `workspace/<task-id>/final_impl.md` per [delivering-go-task-end-to-end](file:///Users/apple/Documents/Github/codetok/.claude/skills/delivering-go-task-end-to-end/SKILL.md) section 3.

## Standardized Delivery Flow (reference)

Every claimed task follows the Stage Loop from [delivering-go-task-end-to-end](file:///Users/apple/Documents/Github/codetok/.claude/skills/delivering-go-task-end-to-end/SKILL.md). Summary for fast lookup:

1. **Claim & isolate**: create git worktree, update this board, create `workspace/<task-id>/`.
2. **Clarify implementation**: write `original_impl.md` -> `new_impl.md` -> `final_impl_v1.md` -> reviewed -> `final_impl.md`.
3. **Spec & test strategy**: if behavior changes, create OpenSpec change via `openspec-ff-change`; write `test_strategy.md`.
4. **TDD implementation**: minimal increments, owner integrates if multiple agents.
5. **Verify**: `make fmt`, `make vet`, `make test`, `make build`, `make lint` (when available); add focused CLI/e2e smoke as relevant to the task.
6. **Code review**: request review, triage findings; bugs fix now, scope-creep into `workspace/<task-id>/todo.md`.
7. **Close**: archive OpenSpec change if applicable, mark task `done` here, record evidence.

Trivial tasks (single-file, no behavior change, e.g. ENG-005) may collapse stages 2 and 3 into a one-line `final_impl.md` and skip multi-agent review, but **must** still run the named verification gate and append a `Change Log` entry.

## Change Log

- 2026-06-13 01:30 CST: Initialized roadmap from current `origin/main` state. Catalogued 10 engineering tasks (ENG-001..010) covering observability of read failures, Windows CI parity, local `make ci` gate, four follow-ups from the accepted 2026-04-15 AI-agent CLI standard (provider validation, token legend, `sources` command), GitHub governance templates, repo-wide secret scan baseline, contributor onboarding doc, and release-rehearsal SOP. Verified that all event-based-token-aggregation (EBTA-001..010) and cursor-usage-support tasks are already `done` and intentionally excluded from this roadmap.
- 2026-06-15 CST: Re-materialized this file after detecting the previous write had not persisted to disk (the 2026-06-13 session's file was missing from `docs/plans/` when the user checked); content unchanged from initial draft.
- 2026-06-15 CST: Trae claimed `ENG-002` (Windows CI parity) with workspace `workspace/ENG-002/`; no hard dependencies; `Change=-` (CI tooling only).
