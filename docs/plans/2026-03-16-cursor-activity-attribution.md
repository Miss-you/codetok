# Cursor Activity Attribution Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a dedicated `codetok cursor activity` command that reads Cursor's local SQLite tracking database and reports separate `composer` and `tab` activity metrics without affecting token reports.

**Architecture:** Add a dedicated activity reader under the top-level `cursor` package, backed by a pure-Go SQLite driver and a separate activity result model. Expose the reader through `cursor.Service` and wire a new `cursor activity` Cobra subcommand with JSON and table output while keeping `daily` and `session` unchanged.

**Tech Stack:** Go 1.21, Cobra, `database/sql`, pure-Go SQLite driver, existing `make` validation flow

---

### Task 1: Add the dedicated Cursor activity reader

**Files:**
- Create: `cursor/activity.go`
- Create: `cursor/activity_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Write the failing tests**

Add tests that create temporary SQLite databases with a `scored_commits` table and verify:

- dual-category aggregation returns separate `composer` and `tab` counts
- missing database returns a no-data result without error
- composer-only data keeps `tab` at zero
- tab-only data keeps `composer` at zero

**Step 2: Run test to verify it fails**

Run: `go test ./cursor -run 'TestReadActivity' -v`
Expected: FAIL because the activity reader and model do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- a dedicated activity model with non-token field names
- default DB path resolution for `~/.cursor/ai-tracking/ai-code-tracking.db`
- SQLite query against `scored_commits`
- no-data handling for missing/unreadable DBs

**Step 4: Run test to verify it passes**

Run: `go test ./cursor -run 'TestReadActivity' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cursor/activity.go cursor/activity_test.go go.mod go.sum
git commit -m "feat: add cursor activity reader"
```

### Task 2: Expose the activity command

**Files:**
- Modify: `cmd/cursor.go`
- Modify: `cmd/cursor_test.go`
- Possibly modify: `cursor/activity.go`

**Step 1: Write the failing tests**

Add command tests that verify:

- `cursor` registers an `activity` subcommand
- `codetok cursor activity --json --db-path <path>` emits JSON activity data
- default table output uses activity wording, not token wording

**Step 2: Run test to verify it fails**

Run: `go test ./cmd -run 'TestCursor.*Activity' -v`
Expected: FAIL because the command and service method do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- `Activity(context.Context, string)` on the Cursor command service
- `cursor activity` Cobra command
- `--json` and `--db-path` flags
- table and JSON output using the dedicated activity model

**Step 4: Run test to verify it passes**

Run: `go test ./cmd -run 'TestCursor.*Activity' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/cursor.go cmd/cursor_test.go
git commit -m "feat: add cursor activity command"
```

### Task 3: Add regression coverage and docs

**Files:**
- Modify: `e2e/e2e_test.go`
- Modify: `README.md`
- Modify: `README_zh.md`

**Step 1: Write the failing tests**

Add end-to-end coverage that verifies:

- `cursor activity --json` works against a test SQLite DB
- `daily --json` token fields remain unchanged when Cursor activity exists separately
- `session --json` token fields remain unchanged when Cursor activity exists separately

**Step 2: Run test to verify it fails**

Run: `go test ./e2e -run 'TestCursor.*Activity|Test.*Cursor.*NotPollute' -v`
Expected: FAIL because the command and fixtures are not wired yet.

**Step 3: Write minimal implementation**

Update docs to describe:

- the new `cursor activity` command
- that activity attribution is line-based, not token-based
- that `daily` and `session` remain token-only reports

**Step 4: Run test to verify it passes**

Run: `go test ./e2e -run 'TestCursor.*Activity|Test.*Cursor.*NotPollute' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add e2e/e2e_test.go README.md README_zh.md
git commit -m "test: cover cursor activity attribution"
```

### Task 4: Run repository validation and final delivery

**Files:**
- Modify: `openspec/changes/cursor-usage-support/tasks.md`

**Step 1: Check task completion against scope**

Confirm `4.1`, `4.2`, `4.3`, `4.4`, `5.1`, and `5.4` are satisfied by code and tests.

**Step 2: Run repository validation**

Run:

- `make fmt`
- `make vet`
- `make test`
- `make lint`
- `make build`

Expected: all commands pass, with `make lint` allowed to skip only if `golangci-lint` is not installed.

**Step 3: Run manual CLI verification**

Run:

- `./bin/codetok cursor activity --json`

Expected: valid JSON activity output or a clear no-data state from the local tracking DB.

**Step 4: Update change tracking**

Mark completed OpenSpec tasks in `openspec/changes/cursor-usage-support/tasks.md`.

**Step 5: Commit**

```bash
git add openspec/changes/cursor-usage-support/tasks.md
git commit -m "docs: mark cursor activity tasks complete"
```
