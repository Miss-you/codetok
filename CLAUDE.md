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
- `cmd/` — cobra CLI commands (root, daily, session, version)
- `provider/` — `provider.go` defines the `Provider` interface and shared types (`TokenUsage`, `SessionInfo`, `DailyStats`)
- `provider/kimi/` — Kimi CLI parser: reads `wire.jsonl`, extracts `StatusUpdate` token usage
- `stats/` — aggregation logic (group by day, filter by date range)
- `e2e/` — end-to-end tests that build the binary and test CLI output

### Key Design Decisions

- **Provider interface**: each AI tool has its own parser package under `provider/`. They all implement `provider.Provider` with `Name()` and `CollectSessions(baseDir)`.
- **Token aggregation**: `StatusUpdate` events in wire.jsonl contain per-request token counts. We sum them per session, then aggregate by day.
- **No external dependencies beyond cobra**: keep the binary small and dependency-free.

### Data Flow

```
~/.kimi/sessions/**/wire.jsonl
    → kimi.Provider.CollectSessions()
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
2. Add unit tests in `provider/<name>/parser_test.go` with `testdata/` fixtures
3. Wire into `cmd/daily.go` and `cmd/session.go` (instantiate provider + merge sessions)
4. Add e2e test fixtures under `e2e/testdata/`

## Release

Tag with `vX.Y.Z` to trigger the GoReleaser GitHub Action. Binaries are built for linux/darwin/windows on amd64/arm64.
