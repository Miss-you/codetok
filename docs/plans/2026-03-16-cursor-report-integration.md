# Cursor Report Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Unify Cursor report collection across `daily` and `session`, merge default local Cursor CSV sources, preserve legacy layouts, and keep report commands local-only.

**Architecture:** Add a shared collector in `cmd/` that both report commands use. Keep Cursor-specific source resolution in `provider/cursor`, where the default root scans legacy files plus `imports/` and `synced/`, while explicit directory overrides stay authoritative.

**Tech Stack:** Go, Cobra, standard library filesystem helpers, existing e2e binary tests

---

### Task 1: Shared Report Collection

**Files:**
- Create: `cmd/collect.go`
- Create: `cmd/collect_test.go`
- Modify: `cmd/daily.go`
- Modify: `cmd/session.go`

**Step 1: Write the failing test**

Add tests that prove:
- provider-specific `*-dir` flags override `--base-dir`
- missing provider directories are skipped
- non-`os.IsNotExist` provider errors still fail the command path

**Step 2: Run test to verify it fails**

Run: `go test ./cmd -run 'TestCollectSessions'`
Expected: FAIL because the shared collector does not exist yet.

**Step 3: Write minimal implementation**

Add a shared helper in `cmd/collect.go` and replace duplicated collection loops in `runDaily` and `runSession`.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd -run 'TestCollectSessions'`
Expected: PASS.

### Task 2: Cursor Default Source Discovery

**Files:**
- Modify: `provider/cursor/parser.go`
- Modify: `provider/cursor/parser_test.go`

**Step 1: Write the failing test**

Add tests that prove:
- default Cursor root merges root-level legacy CSVs with `imports/` and `synced/`
- missing `imports/` or `synced/` does not fail
- explicit base directory uses only the provided directory

**Step 2: Run test to verify it fails**

Run: `go test ./provider/cursor -run 'TestCollectSessions_(DefaultRoot|ExplicitDir)'`
Expected: FAIL because the current provider recursively scans only one directory root.

**Step 3: Write minimal implementation**

Add source-discovery helpers in `provider/cursor/parser.go` for default-root and explicit-root traversal.

**Step 4: Run test to verify it passes**

Run: `go test ./provider/cursor -run 'TestCollectSessions_(DefaultRoot|ExplicitDir)'`
Expected: PASS.

### Task 3: Acceptance Coverage

**Files:**
- Modify: `e2e/e2e_test.go`

**Step 1: Write the failing test**

Add e2e tests for:
- import-only
- sync-only
- import + sync + legacy coexistence
- sync failure fallback via invalid synced CSV plus valid cached CSV
- explicit `--cursor-dir` override
- no implicit remote access with proxy trap for `daily` and `session`

**Step 2: Run test to verify it fails**

Run: `go test ./e2e -run 'Test(Cursor|Daily|Session).*Cursor'`
Expected: FAIL until the shared collector and Cursor source resolution changes are in place.

**Step 3: Write minimal implementation**

Use temporary test directories and helper functions to build local Cursor fixture layouts without touching real user directories.

**Step 4: Run test to verify it passes**

Run: `go test ./e2e -run 'Test(Cursor|Daily|Session).*Cursor'`
Expected: PASS.

### Task 4: Docs and Help Text

**Files:**
- Modify: `cmd/daily.go`
- Modify: `cmd/session.go`
- Modify: `README.md`
- Modify: `README_zh.md`

**Step 1: Write the failing test**

Use existing help-output expectations only if needed; otherwise verify by targeted content checks after editing docs/help strings.

**Step 2: Write minimal implementation**

Update flag help text and README sections to state:
- default Cursor reporting reads local legacy/imported/synced CSV files
- `--cursor-dir` scans only the provided local directory
- `daily` and `session` do not trigger implicit sync

**Step 3: Run targeted verification**

Run:
- `go test ./cmd`
- `rg -n "implicit sync|imports|synced|cursor-dir" README.md README_zh.md cmd/daily.go cmd/session.go`

Expected: PASS with updated text present.

### Task 5: Repository Validation

**Files:**
- Modify: any touched files above

**Step 1: Run formatting**

Run: `make fmt`
Expected: PASS.

**Step 2: Run unit and e2e tests**

Run: `make test`
Expected: PASS.

**Step 3: Run static checks**

Run:
- `make vet`
- `make lint`

Expected: PASS.

**Step 4: Build and manual CLI verification**

Run:
- `make build`
- `./bin/codetok daily --help`
- `./bin/codetok session --help`

Expected: PASS, with updated Cursor help text visible.
