# Cursor Token Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Cursor token support by parsing local Cursor usage CSV exports from disk, while clearly documenting that Tab token usage is not supported.

**Architecture:** Implement a new `cursor` provider that scans a local directory for exported CSV files, parses each row into a session-like record, and feeds those records into the existing session and daily aggregation flows. Keep the scope local-only: no Cursor API calls, no sync command, and no Tab token inference from Cursor app logs.

**Tech Stack:** Go, Cobra CLI, existing provider registry, table/JSON outputs, Go unit tests and e2e tests.

---

### Task 1: Cursor provider parser

**Files:**
- Create: `provider/cursor/parser.go`
- Create: `provider/cursor/parser_test.go`
- Create: `provider/cursor/testdata/usage-export.csv`

**Step 1: Write the failing tests**

Add tests that define the expected CSV behavior:
- Parse Cursor export headers into `provider.SessionInfo`.
- Map `Input (w/o Cache Write)` to `InputOther`.
- Map `Input (w/ Cache Write)` to `InputCacheCreate`.
- Map `Cache Read` and `Output Tokens`.
- Ignore malformed rows instead of failing the full file.
- Support collecting multiple CSV files from a directory.

**Step 2: Run test to verify it fails**

Run: `go test ./provider/cursor -run Test`
Expected: FAIL because the provider package does not exist yet.

**Step 3: Write minimal implementation**

Implement:
- Provider registration with name `cursor`.
- Default directory resolution for imported Cursor CSV files.
- Directory scan for `.csv` files.
- CSV parsing using standard library `encoding/csv`.
- Per-row `SessionInfo` generation with:
  - `ProviderName = "cursor"`
  - `ModelName` from `Model`
  - `SessionID` derived from file name plus row index or timestamp
  - `StartTime` and `EndTime` from `Date`
  - `Title` describing the Cursor usage kind/model row
  - `Turns = 1`

**Step 4: Run test to verify it passes**

Run: `go test ./provider/cursor -run Test`
Expected: PASS.

### Task 2: CLI wiring and end-to-end coverage

**Files:**
- Modify: `cmd/daily.go`
- Modify: `cmd/session.go`
- Modify: `cmd/root.go`
- Modify: `e2e/e2e_test.go`
- Create: `e2e/testdata/cursor/usage-export.csv`

**Step 1: Write the failing tests**

Add coverage for:
- `--cursor-dir` flag on `daily` and `session`.
- Cursor appearing in JSON and dashboard output.
- Cursor working alongside existing providers without changing default grouping behavior.

**Step 2: Run test to verify it fails**

Run: `go test ./cmd ./e2e -run 'Cursor|Daily|Session'`
Expected: FAIL because the flag/import/provider wiring is incomplete.

**Step 3: Write minimal implementation**

Implement:
- Blank import for `provider/cursor`.
- New `--cursor-dir` flag in both commands.
- Any help text updates needed so Cursor shows up consistently.
- E2E fixture plumbing that isolates Cursor data the same way other providers are isolated.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd ./e2e -run 'Cursor|Daily|Session'`
Expected: PASS.

### Task 3: Documentation and limitation statement

**Files:**
- Modify: `README.md`
- Modify: `README_zh.md`

**Step 1: Write the failing test/check**

Define the doc acceptance criteria:
- Cursor moves from planned to supported.
- The docs explain that Cursor support reads exported CSV files from disk.
- The docs clearly state that Cursor Tab token usage is not counted.

**Step 2: Run check to verify it fails**

Run: `rg -n "Tab token|Cursor" README.md README_zh.md`
Expected: current wording does not yet describe the new limitation accurately.

**Step 3: Write minimal documentation changes**

Update both READMEs with:
- Supported provider list entry for Cursor.
- `--cursor-dir` flag docs.
- Example usage for local Cursor CSV imports.
- Clear limitation note: Cursor Tab token usage is unsupported because the imported export does not provide a defensible Tab token split.

**Step 4: Run check to verify it passes**

Run: `rg -n "Tab token|cursor-dir|Cursor" README.md README_zh.md`
Expected: PASS with the new support and limitation text present.

### Task 4: Validation and review handoff

**Files:**
- Review modified files only.

**Step 1: Run formatting and static validation**

Run:
- `make fmt`
- `make vet`

Expected: PASS.

**Step 2: Run tests and build**

Run:
- `make test`
- `make lint`
- `make build`

Expected: PASS.

**Step 3: Manual CLI smoke check**

Run:
- `EMPTY_DIR="$(mktemp -d)"`
- `./bin/codetok daily --all --cursor-dir "$(pwd)/e2e/testdata/cursor" --kimi-dir "$EMPTY_DIR" --claude-dir "$EMPTY_DIR" --codex-dir "$EMPTY_DIR"`
- `./bin/codetok session --cursor-dir "$(pwd)/e2e/testdata/cursor" --kimi-dir "$EMPTY_DIR" --claude-dir "$EMPTY_DIR" --codex-dir "$EMPTY_DIR"`

Expected:
- Daily dashboard shows Cursor totals.
- Session output shows Cursor rows.

**Step 4: Self-review and handoff**

Check:
- No remote Cursor API calls were added.
- Existing provider behavior is unchanged.
- README limitation text is explicit about unsupported Tab token counting.
