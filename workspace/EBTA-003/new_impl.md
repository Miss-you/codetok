# EBTA-003 Codex Native Usage Events Implementation Plan

**Goal:** Teach the Codex provider to emit native timestamped usage events from `token_count` records, while preserving the existing session parser and adding `CODEX_HOME`-aware default source resolution.

**Architecture:** Keep all changes inside `provider/codex/parser.go` and `provider/codex/parser_test.go`. Add a new event collector alongside the current session collector, then have both collectors share the same directory discovery and Codex home resolution. The event parser should be file-local and conservative: emit one `provider.UsageEvent` per token delta, keep session metadata stable per file, and leave command aggregation for later tasks.

**Tech Stack:** Go, the existing Codex JSONL parser, `os.LookupEnv`, `filepath.Join`, and the current `provider` package types.

---

### Task 1: Add Codex usage-event tests first

**Files:**
- Modify: `provider/codex/parser_test.go`

**Step 1: Write failing tests**

Add focused tests that prove the event parser behavior without changing command code:

- `TestParseCodexUsageEvents_LastTokenUsageEmitsOneEvent`
- `TestParseCodexUsageEvents_TotalTokenUsageSubtractsPreviousTotals`
- `TestParseCodexUsageEvents_PreservesOriginalTimestamps`
- `TestParseCodexUsageEvents_UsesTurnContextModelFallback`
- `TestCollectCodexSessions_UsesCodexHomeWhenBaseDirEmpty`
- `TestCollectCodexUsageEvents_UsesCodexHomeWhenBaseDirEmpty`

Use small JSONL fixtures in test bodies or `testdata/` files. The important assertions are:

- `last_token_usage` produces a single event with that exact usage.
- cumulative `total_token_usage` lines produce deltas by subtracting the previous total.
- the emitted event timestamp is the `token_count` line timestamp, not the session start time.
- model lookup falls back to `turn_context.payload.model` when the token record has no explicit model.
- empty `baseDir` respects `CODEX_HOME` before the `~/.codex` fallback.

**Step 2: Run the narrow test set**

Run:

```bash
go test ./provider/codex -run 'Test(ParseCodexUsageEvents|CollectCodexUsageEvents|CodexHome)'
```

Expected: fail because the event collector and `CODEX_HOME` handling do not exist yet.

---

### Task 2: Add the Codex event collector and parser helpers

**Files:**
- Modify: `provider/codex/parser.go`
- Modify: `provider/codex/parser_test.go`

**Step 1: Implement the minimal parser surface**

Add these functions:

- `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)`
- `func parseCodexUsageEvents(path string) ([]provider.UsageEvent, error)`
- `func resolveCodexBaseDir(baseDir string) (string, error)` or an equivalent helper shared by both collectors
- `func collectCodexSessionPaths(baseDir string) ([]string, error)` or equivalent shared discovery code

Keep `CollectSessions` intact, but make both collectors call the same base-dir and file-discovery helpers so the directory resolution logic stays consistent.

**Step 2: Implement delta handling**

The event parser should follow these rules:

- Prefer `last_token_usage` when present.
- If `last_token_usage` is missing, compute the delta from `total_token_usage` minus the previous cumulative total in the same file.
- Update the previous cumulative total whenever `total_token_usage` exists.
- Skip zero-delta events.
- Treat `cached_input_tokens` as input by computing `InputOther = input_tokens - cached_input_tokens`.
- Do not add `reasoning_output_tokens` to `Output`; Codex output already includes it.

This task should not change how `parseCodexSession` works. The session collector remains the compatibility path for callers that still need `provider.SessionInfo`.

**Step 3: Preserve the same model and session metadata on each event**

Populate each `provider.UsageEvent` with the stable metadata already available in the JSONL file:

- `ProviderName`: `codex`
- `SessionID`: prefer `session_meta.id`
- `Title`: keep the first useful user-facing title, usually the first `user_message`
- `WorkDirHash`: preserve any existing session/workdir identifier if Codex exposes one; otherwise leave it empty
- `ModelName`: keep the latest non-placeholder model seen in the file
- `SourcePath`: the session file path
- `EventID`: a stable per-event identifier only if one already exists in the record; otherwise leave it empty

Model extraction should keep the current raw JSON fallback chain and extend it for event parsing:

- `turn_context.payload.model`
- `event_msg.payload.model`
- `event_msg.payload.info.model`
- existing raw JSON model paths already used by `extractModelFromRawJSON`

**Step 4: Add the `CODEX_HOME` resolution behavior**

Use this default source order inside the provider:

1. explicit `baseDir` argument, when non-empty
2. `$CODEX_HOME/sessions`, when `CODEX_HOME` is set
3. `~/.codex/sessions`

Apply this same resolution for both `CollectSessions` and `CollectUsageEvents`. The key point is that `CODEX_HOME` only changes the provider’s local file root; it must not introduce any remote API behavior.

**Step 5: Re-run the Codex package tests**

Run:

```bash
go test ./provider/codex
```

Expected: pass, with the existing session parser tests still green and the new usage-event tests green.

---

### Task 3: Validate the narrow change set

**Files:**
- No new files beyond the two modified Codex parser files

**Step 1: Run the focused command set**

Run:

```bash
go test ./provider/codex -run 'Test(ParseCodexUsageEvents|CollectCodexUsageEvents|CodexHome)'
go test ./provider/codex
```

Expected: both commands pass.

**Step 2: Keep the scope closed**

Do not touch `cmd/`, `stats/`, or the other providers in this task. EBTA-003 only needs the Codex native event producer and its tests; command-level aggregation and cross-provider rollout belong to the later tasks on the board.
