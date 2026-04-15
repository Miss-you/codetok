# AI-Agent-Friendly CLI UX Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `codetok` easier for humans and AI agents to discover, run, and correctly interpret without reading source code.

**Architecture:** Keep the existing Cobra command structure and local-only reporting boundary. Improve help text, validation, table semantics, and source observability at the CLI layer while preserving provider parsers and token aggregation semantics.

**Tech Stack:** Go, Cobra, existing provider registry, existing command/e2e tests

---

## Context

This plan captures a CLI UX review of `codetok v0.4.0` after manually running:

- `codetok --help`
- `codetok daily --help`
- `codetok session --help`
- `codetok cursor --help`
- `codetok daily --provider codex --since 2026-04-15 --unit raw`
- `codetok session --provider codex --since 2026-04-15`
- invalid argument probes such as `--provider nope` and `--group-by invalid`

The numeric token counts were cross-checked against Codex local JSONL session logs and were accurate relative to local Codex `token_count` records. The remaining problem is not counting accuracy; it is whether a first-time human or AI agent can understand the command surface and token semantics from the CLI alone.

## SOP: Reviewing AI-Agent-Friendly CLI Suggestions

Use this SOP before accepting CLI UX changes as product standards.

1. **Run the CLI as a first-time user**
   - Capture top-level help, command help, common default output, JSON output, and representative error output.
   - Do not read source first; the first pass should reflect what a user or agent sees.

2. **Map observed confusion to a concrete standard**
   - Classify the issue as discoverability, semantic clarity, diagnostics, source/scope transparency, side-effect safety, or backward-compatible ergonomics.
   - Reject suggestions that are only stylistic unless they reduce a real mistake or automation failure.

3. **Verify against source and docs**
   - Inspect command definitions, output formatting, provider registry behavior, README claims, and tests.
   - Confirm whether the issue is real, already documented elsewhere, or caused by stale binary output.

4. **Assign a verdict**
   - **Keep:** The suggestion should become part of the agent-friendly CLI standard as proposed.
   - **Modify:** The underlying problem is valid, but the implementation should be narrower, renamed, or reframed.
   - **Drop:** The suggestion does not improve correctness, discoverability, or automation enough to justify CLI surface area.

5. **Use independent second review for standards**
   - Split unrelated suggestions across subagents.
   - Require each subagent to return: item, verdict, rationale, and acceptance criterion.
   - Merge the final decision only after reconciling subagent feedback with repo evidence.

6. **Write acceptance criteria before implementation**
   - Each accepted item must state exact command behavior, help text expectation, output column semantics, and test target.
   - Prefer tests that assert behavior over snapshotting large terminal output.

7. **Preserve compatibility by default**
   - Add aliases rather than renaming commands.
   - Keep JSON fields stable unless the change is explicitly a schema migration.
   - Use clearer labels in human output when raw field names would be confusing.

## Standard: AI-Agent-Friendly CLI Criteria

`codetok` CLI changes should satisfy these criteria.

1. **Self-discovery:** `--help` must show the shortest useful path from no context to a valid command.
2. **Semantic consistency:** The same label must mean the same thing across human tables, JSON, README examples, and help legends.
3. **Metric transparency:** Aggregated counters must explain their components, especially cache and total formulas.
4. **Actionable diagnostics:** Invalid user input must fail loudly with allowed values and avoid making typos look like empty data.
5. **Machine-readable parity:** JSON output must expose the same underlying facts as human output, even when labels differ for readability.
6. **Source transparency:** Local scanners must show or document which data roots are scanned and when data is unavailable.
7. **Side-effect clarity:** Commands must state whether they read local files only or may contact a remote API.
8. **Snapshot honesty:** Reports over active local logs must state that results are point-in-time snapshots.
9. **Backward-compatible ergonomics:** More natural aliases are welcome when they do not remove or redefine existing commands.
10. **Concise failure mode:** Errors should be specific and brief by default; full help is for `--help`.

## Second-Review Summary

Three subagents independently reviewed the original ten suggestions:

- **Faraday:** reviewed items 1-4. Result: keep items 1 and 4; modify items 2 and 3.
- **Plato:** reviewed items 5-7. Result: keep items 5 and 6; modify item 7 to prefer `sources` over `doctor`.
- **Darwin:** reviewed items 8-10. Result: keep all three.

No suggestion was dropped. Three suggestions were narrowed before acceptance: date help wording, input label semantics, and source-inspection command naming.

## Reviewed Improvement Backlog

| # | Final verdict | Standard fit | What to do | Why | Second-review result | Acceptance criterion |
|---|---|---|---|---|---|---|
| 1 | Keep | Self-discovery | Add a short `Examples:` section to top-level `codetok --help`. Include `daily`, `session`, JSON, and local Cursor examples. | A first-time user or agent should not need the README to discover the common path. | Faraday: keep. Top-level help is currently minimal while README already has quick-start material. | `codetok --help` shows 3-5 runnable examples without hiding existing command list. |
| 2 | Modify | Self-discovery, diagnostics | Replace user-facing `format: 2006-01-02` wording with `YYYY-MM-DD`, plus an example such as `2026-04-15`. Keep Go parsing unchanged. | Go's reference date is correct in code but opaque in CLI help. | Faraday: modify. The idea is right, but it should be framed as human wording, not parser behavior. | `daily --help` and `session --help` describe date flags as `YYYY-MM-DD, e.g. 2026-04-15`. |
| 3 | Modify | Semantic consistency, metric transparency | Define one human-table label policy for input fields. Prefer explicit labels such as `Input Total`, `Input Other`, `Cache Read`, and `Cache Create`; avoid bare `Input` when it means different things. | `daily` currently labels `InputOther` as `Input`, while `session` labels `TotalInput()` as `Input`. That makes accurate data look suspect. | Faraday: modify. The mismatch is real; the fix is a label policy and formula documentation. | Human tables and README examples use consistent labels; tests prove `daily` and `session` do not assign different meanings to the same label. |
| 4 | Keep | Metric transparency | Add a compact token-field legend to `daily --help`, `session --help`, README, and README_zh. | Users need to know whether total includes cache reads, cache creation, and output. | Faraday: keep. The token model exists in `provider.TokenUsage` but is not visible enough from command help. | Help explains `input_other`, `input_cache_read`, `input_cache_creation`, `output`, `input_total`, and `total`. |
| 5 | Keep | Semantic consistency, machine-readable parity | Show cache breakdown in `codetok session` human output by default, unless a later implementation proves the table is too wide and adds an explicit `--breakdown` instead. | Cache is first-class data and can dominate totals; hiding it in session output causes the same trust problem users ask about. | Plato: keep. README_zh already documents cache columns for session-like output, so this is a visibility gap. | A session fixture with cache usage shows `Cache Read` and `Cache Create` in human output, while JSON remains unchanged. |
| 6 | Keep | Actionable diagnostics | Validate `--provider` against the provider registry before collection. Unknown provider values should fail with allowed names. | A typo like `--provider codxe` currently looks like "no data", which is bad for both humans and agents. | Plato: keep. `collectSessionsFromProviders` filters to an empty provider list and silently succeeds. | `codetok daily --provider bogus` and `codetok session --provider bogus` exit non-zero and list valid providers; a valid provider with no files still succeeds with empty data. |
| 7 | Modify | Source transparency, side-effect clarity | Add `codetok sources` rather than `doctor`. It should show resolved scan roots, existence, provider names, and discovered file/session counts without remote calls. | Users need a way to answer "what did the tool actually scan?" before trusting totals. `sources` is more precise than `doctor`. | Plato: modify. The capability is valuable, but `doctor` is vague. | `codetok sources` reports each provider's local roots, whether they exist, and local discovery counts; it performs no network access. |
| 8 | Keep | Backward-compatible ergonomics | Add `sessions` as an alias for `session`. Keep `session` canonical. | The command reports many sessions, and plural is a natural guess. | Darwin: keep. This improves discoverability without breaking existing scripts. | `codetok sessions` accepts the same flags and runs the same code path; help lists it as an alias. |
| 9 | Keep | Concise failure mode | Configure command error handling so validation failures do not print duplicate error text or full usage by default. | Agents parse errors better when output has one clear failure and one optional next step. | Darwin: keep. Existing validation errors are specific, but Cobra usage noise can obscure them. | Invalid values print one specific error line and a short help hint, not duplicated errors or full command help. |
| 10 | Keep | Snapshot honesty | Document that active sessions may change while `daily` and `session` run. Put the warning in README and command help where it will not dominate output. | `codetok` reads local logs that active CLIs may still be writing. This is accurate, not a bug. | Darwin: keep. Existing docs mention local-file scope, but not active-write snapshot behavior. | Docs/help state reports are point-in-time local snapshots and active sessions may change between runs. |

## Implementation Plan

### Task 1: Help Text and Examples

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/daily.go`
- Modify: `cmd/session.go`
- Modify: `README.md`
- Modify: `README_zh.md`

**Steps:**
1. Add top-level examples to `rootCmd.Long`.
2. Rewrite date flag descriptions to use `YYYY-MM-DD` examples.
3. Add a compact token-field legend to `dailyCmd.Long` and `sessionCmd.Long`.
4. Add README sections that mirror the help text without introducing new behavior.
5. Verify with `go test ./cmd` and `go run . --help`.

### Task 2: Consistent Token Labels

**Files:**
- Modify: `cmd/daily.go`
- Modify: `cmd/session.go`
- Modify: `cmd/daily_test.go`
- Modify: `cmd/session_test.go` or create it if absent
- Modify: `README.md`
- Modify: `README_zh.md`

**Steps:**
1. Define the final label policy for human output.
2. Update `daily` share-table headers so `InputOther` is not shown as ambiguous `Input`.
3. Update `session` table headers and totals to show cache fields clearly.
4. Add tests for table headers and totals using fixture `provider.SessionInfo` values.
5. Verify JSON output remains compatible.

### Task 3: Provider Validation and Errors

**Files:**
- Modify: `cmd/collect.go`
- Modify: `cmd/collect_test.go`
- Modify: `cmd/root.go` if Cobra error behavior is centralized there
- Modify: e2e tests if existing coverage expects full usage dumps

**Steps:**
1. Add provider-filter validation against `provider.Registry()`.
2. Return a clear error with valid provider names when the filter is unknown.
3. Configure Cobra to avoid duplicate validation errors and unnecessary full usage dumps.
4. Add tests for unknown providers and valid providers with no data.
5. Manually verify `--group-by invalid`, `--unit invalid`, and `--provider invalid` outputs.

### Task 4: Source Inventory Command

**Files:**
- Create: `cmd/sources.go`
- Create: `cmd/sources_test.go`
- Modify: provider interfaces only if absolutely needed
- Modify: `README.md`
- Modify: `README_zh.md`

**Steps:**
1. Design `codetok sources` output around local roots, existence, provider name, and discovered counts.
2. Reuse provider registry and directory override flag conventions.
3. Keep the command local-only and avoid implicit Cursor sync or auth checks.
4. Add tests with temporary provider roots.
5. Verify `codetok sources`, `codetok sources --provider codex`, and missing-directory cases.

### Task 5: Session Alias and Snapshot Notice

**Files:**
- Modify: `cmd/session.go`
- Modify: `README.md`
- Modify: `README_zh.md`
- Modify: e2e tests if command discovery is covered

**Steps:**
1. Add `Aliases: []string{"sessions"}` to `sessionCmd`.
2. Ensure alias help does not duplicate command documentation as a separate command.
3. Add active-session snapshot wording to help/docs.
4. Verify `go run . sessions --help` and `go run . session --help`.

### Task 6: Repository Validation

**Files:**
- All modified files above

**Steps:**
1. Run `make fmt`.
2. Run `make test`.
3. Run `make vet`.
4. Run `make lint` if available.
5. Run `make build`.
6. Manually verify:
   - `./bin/codetok --help`
   - `./bin/codetok daily --help`
   - `./bin/codetok session --help`
   - `./bin/codetok sessions --help`
   - `./bin/codetok sources --help`
   - representative invalid flag outputs

## Non-Goals

- Do not change token counting algorithms in provider parsers as part of this UX pass.
- Do not change JSON field names unless a separate schema migration is approved.
- Do not add remote API access to `daily`, `session`, or `sources`.
- Do not remove existing `session` command spelling.

## Open Questions Before Implementation

1. Should `session` show all cache columns by default, or should a compact default plus `--breakdown` be introduced if terminal width becomes a problem?
2. Should `sources` count parsed sessions or discovered files first? Counting parsed sessions is more useful, but discovered files can explain parse failures.
3. Should concise error handling be applied globally at `rootCmd`, or only to report commands first?
