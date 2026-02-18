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
- Full history: `codetok daily --all`.
- Custom rolling window: `codetok daily --days N`.
- Explicit range: `codetok daily --since YYYY-MM-DD --until YYYY-MM-DD`.
- Display unit: `codetok daily --unit raw|k|m|g` (default `k`, table output only).

Flag constraints:
- `--all` is mutually exclusive with `--days`, `--since`, `--until`.
- `--days` is mutually exclusive with `--since`, `--until`.

## Architecture Pointers

- `cmd/` — Cobra commands (`daily`, `session`, `version`)
- `provider/` — provider parsers + registry + bounded parallel parser
- `stats/` — date filtering and day-level aggregation
- `e2e/` — end-to-end CLI tests
