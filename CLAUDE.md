# CLAUDE.md

This file provides guidance to Claude Code when working on this project.

## Project Overview

codetok is a Go CLI tool that aggregates token usage from AI coding CLI tools (Kimi CLI, Claude Code, Codex CLI, etc.). It reads local session data files, extracts token counts, and outputs daily/session reports.

## Build & Test

```bash
make build          # Build binary to ./bin/codetok
make test           # Run all tests with -race -cover
make lint           # Run golangci-lint (must be installed)
make fmt            # Format code (go fmt + goimports)
make vet            # Run go vet
make tidy           # go mod tidy + verify
```

Always run `make test` after changes to verify nothing is broken.

## Architecture

### Package Layout

- `main.go` — entrypoint, injects version via ldflags
- `cmd/` — cobra CLI commands (root, daily, session, version); multi-provider with per-provider dir flags
- `provider/provider.go` — `Provider` interface and shared types (`TokenUsage`, `SessionInfo`, `DailyStats`)
- `provider/registry.go` — global provider registry with `Register()` / `Registry()` / `FilterProviders()`; providers self-register via `init()`
- `provider/parallel.go` — `ParseParallel()` helper: bounded goroutine pool with semaphore pattern, default `min(NumCPU, 8)` workers, configurable via `CODETOK_WORKERS` env var
- `provider/kimi/` — Kimi CLI parser: reads `wire.jsonl`, extracts `StatusUpdate` token usage
- `provider/claude/` — Claude Code parser: reads session JSONL, extracts `assistant` message usage with streaming dedup (`messageId:requestId` composite key, last-entry-wins)
- `provider/codex/` — Codex CLI parser: reads rollout JSONL, extracts last cumulative `token_count` event
- `stats/` — aggregation logic (group by day+provider, filter by date range)
- `e2e/` — end-to-end tests that build the binary and test CLI output; uses `isolatedArgs()` to prevent cross-provider interference

### Key Design Decisions

- **Provider auto-registration**: each provider package calls `provider.Register()` in its `init()` function. The CLI imports providers with blank imports (`_ "..."`). No manual wiring needed.
- **Parallel parsing**: `provider.ParseParallel()` uses a buffered channel as semaphore + `sync.Mutex` for result collection. Workers capped at 8 to avoid file I/O contention. Race-safe (tested with `-race`).
- **Claude Code dedup**: streaming causes the same assistant message to appear multiple times with increasing token counts (up to 56% over-counting). Deduplicated using `messageId:requestId` composite key, keeping only the final entry.
- **Codex CLI cumulative tokens**: token counts are cumulative, so we take only the last `token_count` event per session.
- **No external dependencies beyond cobra**: keep the binary small and dependency-free.

### Data Flow

```
~/.kimi/sessions/**/wire.jsonl           → kimi.Provider.CollectSessions()
~/.claude/projects/**/*.jsonl            → claude.Provider.CollectSessions()  (with dedup)
~/.codex/sessions/**/*.jsonl             → codex.Provider.CollectSessions()
    ↓ (all parsed in parallel via ParseParallel)
    → []provider.SessionInfo
    → stats.FilterByDateRange()
    → stats.AggregateByDay()
    → cmd prints table or JSON
```

## Coding Conventions

- Standard Go project layout
- Use `text/tabwriter` for table output (no external table libraries)
- JSON output uses `json:"snake_case"` tags matching the source data field names
- Error handling: wrap with `fmt.Errorf("context: %w", err)`, skip malformed data gracefully
- Tests: use `testdata/` directories for fixtures, test both valid and edge cases
- Imports: stdlib first, then external, then internal (enforced by goimports with local-prefixes)

## Adding a New Provider

1. Create `provider/<name>/parser.go` implementing `provider.Provider`
2. Call `provider.Register(&Provider{})` in `init()` for auto-registration
3. Use `provider.ParseParallel()` for concurrent file parsing
4. Add unit tests in `provider/<name>/parser_test.go` with `testdata/` fixtures
5. Add blank import in `cmd/daily.go` and `cmd/session.go`: `_ "github.com/Miss-you/codetok/provider/<name>"`
6. Add `--<name>-dir` flag if the provider needs a custom data directory override
7. Add e2e test fixtures under `e2e/testdata/` and update `isolatedArgs()` in `e2e/e2e_test.go`

## Release

Tag with `vX.Y.Z` to trigger the GoReleaser GitHub Action. Binaries are built for linux/darwin/windows on amd64/arm64.
