# AGENTS.md

This file provides guidance to coding agents working in this repository.

## Project Summary

codetok is a Go CLI that aggregates token usage from local session logs produced by coding assistants (Kimi CLI, Claude Code, Codex CLI). It reports daily and per-session usage.

Statistics scope:
- Aggregate token counters from existing local session files.
- Do not call provider remote APIs.
- Count only sessions that still exist on disk.

## Build, Test, Validate

```bash
make build          # Build binary to ./bin/codetok
make test           # Run all tests with -race -cover
make lint           # Run golangci-lint if installed
make fmt            # Format code
make vet            # Run go vet
```

Important:
- Rebuild with `make build` before manual CLI verification with `./bin/codetok`.
- `go run . ...` may show newer behavior than an old `./bin/codetok`.

## Daily Command Contract

- Default: `codetok daily` shows the latest 7-day rolling window.
- Default grouping: `codetok daily` aggregates by CLI/provider (`--group-by cli`).
- Model view (explicit opt-in): `codetok daily --group-by model`.
- Default terminal output layout:
  - `Daily Total Trend` (date-axis trend of total usage)
  - `Model/CLI Total Ranking` (period total ranking by current group)
  - `Top N Model/CLI Share` (share + detailed token split)
- Full history: `codetok daily --all`.
- Custom rolling window: `codetok daily --days N`.
- Explicit range: `codetok daily --since YYYY-MM-DD --until YYYY-MM-DD`.
- Display unit: `codetok daily --unit raw|k|m|g` (default `m`, dashboard output only).
- Share section size: `codetok daily --top N` (default `5`, applies to the current group-by dimension).
- JSON grouping semantics: `provider` keeps provider meaning; grouped dimension and value are described by `group_by` + `group`.

Flag constraints:
- `--all` is mutually exclusive with `--days`, `--since`, `--until`.
- `--days` is mutually exclusive with `--since`, `--until`.

## Architecture Pointers

- `cmd/` — Cobra commands (`daily`, `session`, `version`)
- `provider/` — provider parsers + registry + bounded parallel parser
- `stats/` — date filtering and day-level aggregation
- `e2e/` — end-to-end CLI tests
