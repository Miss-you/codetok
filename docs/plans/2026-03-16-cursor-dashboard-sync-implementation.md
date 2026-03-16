# Cursor Dashboard Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add explicit `codetok cursor login|status|sync|logout` commands that validate and store a Cursor session token locally, fetch dashboard CSV on demand, and atomically persist synced CSVs without changing `daily` or `session` into networked commands.

**Architecture:** Keep `provider/cursor` as a local CSV reader and add a separate `cmd cursor` plus internal Cursor sync package for credentials, HTTP calls, and cache writes. Reuse the existing `~/.codetok/cursor` root, reserve `synced/` for tool-owned CSV artifacts, and make sync/update paths atomic so failed network calls never clobber existing cache.

**Tech Stack:** Go, Cobra, `net/http`, `httptest`, existing provider parser/tests, repo Makefile validation commands.

---

### Task 1: Cursor sync domain package and command skeleton

**Files:**
- Create: `cursor/auth.go`
- Create: `cursor/client.go`
- Create: `cursor/storage.go`
- Create: `cursor/sync.go`
- Create: `cmd/cursor.go`
- Modify: `cmd/root.go`
- Test: `cursor/auth_test.go`
- Test: `cursor/client_test.go`
- Test: `cmd/cursor_test.go`

**Step 1: Write the failing tests**

Add tests covering:
- `codetok cursor` root command advertises `login`, `status`, `sync`, `logout`
- login requires a token input and returns validation failure on invalid token
- status distinguishes `not logged in`, `credential saved but invalid`, and `credential saved and valid`
- logout removes saved credentials and reports logged-out state

**Step 2: Run test to verify it fails**

Run: `go test ./cmd ./cursor -run 'Cursor|Login|Status|Logout'`
Expected: FAIL because the command package and sync package do not exist yet.

**Step 3: Write minimal implementation**

Implement:
- `cmd/cursor.go` with root `cursor` command and subcommands `login`, `status`, `sync`, `logout`
- injectable service dependencies so command tests can stub validation/sync
- user-facing help text that states remote access is explicit to the `cursor` subcommands only

**Step 4: Run test to verify it passes**

Run: `go test ./cmd ./cursor -run 'Cursor|Login|Status|Logout'`
Expected: PASS.

### Task 2: Credential storage with restricted permissions and atomic writes

**Files:**
- Modify: `cursor/storage.go`
- Test: `cursor/storage_test.go`

**Step 1: Write the failing tests**

Add tests covering:
- credentials default to a `codetok`-owned config path under the user home dir
- stored token file permissions are restricted to owner read/write
- writes use temp file + rename semantics
- failed write leaves previous credential file intact

**Step 2: Run test to verify it fails**

Run: `go test ./cursor -run 'Credential|Storage|Atomic'`
Expected: FAIL because storage behavior is not implemented yet.

**Step 3: Write minimal implementation**

Implement:
- path helpers for Cursor root, credential file, and sync directory
- credential load/save/delete methods
- directory creation with secure permissions
- atomic file replacement helper used for credential writes

**Step 4: Run test to verify it passes**

Run: `go test ./cursor -run 'Credential|Storage|Atomic'`
Expected: PASS.

### Task 3: HTTP client for validation and CSV export

**Files:**
- Modify: `cursor/client.go`
- Test: `cursor/client_test.go`

**Step 1: Write the failing tests**

Add `httptest` coverage for:
- valid token status check
- invalid or expired token response
- successful CSV export
- non-CSV response rejection
- network error propagation
- configurable base URL for tests

**Step 2: Run test to verify it fails**

Run: `go test ./cursor -run 'Client|Validate|Export|HTTP'`
Expected: FAIL because the HTTP client does not exist yet.

**Step 3: Write minimal implementation**

Implement:
- Cursor API client with injectable `baseURL` and `http.Client`
- validation call against usage summary/status endpoint
- CSV export call against dashboard export endpoint
- explicit content validation before sync writes

**Step 4: Run test to verify it passes**

Run: `go test ./cursor -run 'Client|Validate|Export|HTTP'`
Expected: PASS.

### Task 4: Sync cache writes and parser compatibility

**Files:**
- Modify: `cursor/sync.go`
- Modify: `provider/cursor/parser.go`
- Test: `cursor/sync_test.go`
- Test: `provider/cursor/parser_test.go`

**Step 1: Write the failing tests**

Add tests covering:
- successful sync writes CSV into `~/.codetok/cursor/synced/`
- sync failure preserves prior cached CSV
- synced CSV is discoverable by the existing Cursor provider without extra flags
- default reporting still does not perform remote calls

**Step 2: Run test to verify it fails**

Run: `go test ./cursor ./provider/cursor -run 'Sync|Synced|Cache|Discover'`
Expected: FAIL because sync writing and default discovery are incomplete.

**Step 3: Write minimal implementation**

Implement:
- atomic sync target write using temporary file in the target directory
- stable synced filename policy
- parser/default-directory behavior that continues to discover root imports plus nested `synced/`
- sync orchestration that validates saved credentials before fetching CSV and never mutates cache on failed export

**Step 4: Run test to verify it passes**

Run: `go test ./cursor ./provider/cursor -run 'Sync|Synced|Cache|Discover'`
Expected: PASS.

### Task 5: Command integration, docs, and validation

**Files:**
- Modify: `README.md`
- Modify: `README_zh.md`
- Modify: `cmd/root.go`
- Modify: `e2e/e2e_test.go`

**Step 1: Write the failing tests/checks**

Add or update checks covering:
- CLI help exposes Cursor auth/sync commands
- docs say `daily` and `session` remain local-only and do not auto-sync
- end-to-end coverage for sync-only cache reuse path where latest sync fails but old CSV still reports

**Step 2: Run test/check to verify it fails**

Run:
- `go test ./e2e -run 'Cursor|Sync'`
- `rg -n "cursor sync|local-only|implicit sync" README.md README_zh.md`

Expected: FAIL until docs/tests match the new behavior.

**Step 3: Write minimal implementation**

Implement:
- README updates for login/status/sync/logout usage and local-only reporting constraint
- e2e acceptance around synced cache visibility and no implicit network sync in report commands

**Step 4: Run test/check to verify it passes**

Run:
- `go test ./e2e -run 'Cursor|Sync'`
- `rg -n "cursor sync|local-only|implicit sync" README.md README_zh.md`

Expected: PASS.

### Task 6: Repository validation and review handoff

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
- `./bin/codetok cursor --help`
- `./bin/codetok cursor status`
- `./bin/codetok daily --all --cursor-dir "$(pwd)/e2e/testdata/cursor"`

Expected:
- Cursor subcommands are visible.
- Status works without crashing when logged out.
- Daily report still reads only local CSV files.

**Step 4: Review and delivery**

Check:
- no remote calls are reachable from `daily` or `session`
- failed sync does not delete or truncate existing CSV cache
- credential storage is local-only with restricted permissions
- PR content and Linear status are updated only after validation passes
