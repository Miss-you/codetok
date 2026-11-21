# ENG-015 Final Implementation

Trivial task (one test file, no product behavior change), so stages 2 and 3 are collapsed here per the roadmap's "Trivial tasks" rule. `Change=-`: only test harness code changes; CLI behavior and output are untouched.

## Before

`e2e/e2e_test.go` `buildBinary(t)` ran `go build -o <t.TempDir()>/codetok ..` for every test, with `GOCACHE` set to a fresh `t.TempDir()`. All 27 test functions recompiled the module and its dependencies (including `modernc.org/sqlite`) from scratch: ~6 s each, ~190 s for `go test -count=1 ./e2e/`. The empty cache also meant CI and self-hosted runner Go caches never helped.

## After

- `TestMain` creates one temp dir (`os.MkdirTemp`) for the binary and removes it after `m.Run()`.
- `buildOnce = sync.OnceValues(...)` runs `go build` once, on first use, with the inherited environment (so the inherited `GOCACHE`). `sync.OnceValues` exists since Go 1.21, which matches `go.mod`.
- `buildBinary(t)` keeps its signature and returns the shared path, failing the calling test with the build output if the build failed. No call sites changed.
- Lazy build: a `-run` subset that needs no binary pays no build cost.

## Isolation

The binary is read-only shared state. Each test still builds its own data dirs (`isolatedArgs()`, `emptyDir`, `t.TempDir()` homes); no test writes next to or modifies the binary (checked with grep for `Dir(bin`, `Chmod`, `os.Remove`).
