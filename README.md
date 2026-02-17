# codetok

[![CI](https://github.com/Miss-you/codetok/actions/workflows/ci.yml/badge.svg)](https://github.com/Miss-you/codetok/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)

[中文文档](README_zh.md)

A CLI tool for tracking and aggregating token usage across AI coding CLI tools.

Currently supported:

- **Kimi CLI** — parses `~/.kimi/sessions/**/wire.jsonl`

Planned:

- Claude Code
- OpenCode
- Codex CLI
- Cursor

## Installation

### From source

```bash
go install github.com/Miss-you/codetok@latest
```

### Build locally

```bash
git clone https://github.com/Miss-you/codetok.git
cd codetok
make build
# Binary at ./bin/codetok
```

### From release

Download pre-built binaries from the [Releases](https://github.com/Miss-you/codetok/releases) page.

## Quick Start

```bash
# Show daily token usage breakdown
codetok daily

# Show per-session token usage
codetok session

# Output as JSON
codetok daily --json

# Filter by date range
codetok daily --since 2026-02-01 --until 2026-02-15
```

## Usage

### `codetok daily`

Show daily token usage breakdown.

```
Date        Sessions  Input    Output  Cache Read  Cache Create  Total
2026-02-07  5         109822   15356   632985      0             758163
2026-02-08  2         95046    7010    274232      0             376288
2026-02-15  21        938566   149287  7869696     0             8957549
TOTAL       49        2965044  369854  24638673    0             27973571
```

Flags:

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--since` | Start date filter (format: `2006-01-02`) |
| `--until` | End date filter (format: `2006-01-02`) |
| `--base-dir` | Override default Kimi data directory |

### `codetok session`

Show per-session token usage.

```
Date        Session                               Title                      Input     Output  Total
2026-02-13  75c64dba-5c10-4717-83cd-f3d33abc39bc  Translate article...       72405     6080    78485
2026-02-15  01f3c3c6-a4df-4e2b-8249-ea045ab13f11  Write documentation...     381667    28258   409925
TOTAL                                                                        2965044   369854  27973571
```

Flags: same as `codetok daily`.

### `codetok version`

Print build version, commit hash, and build date.

## How It Works

codetok reads local session data that AI coding CLIs store on disk:

**Kimi CLI** stores session data at `~/.kimi/sessions/<work-dir-hash>/<session-uuid>/`:

- `wire.jsonl` — event stream with `StatusUpdate` events containing `token_usage`
- `metadata.json` — session title and ID

codetok scans all session directories, extracts token counts from `StatusUpdate` events, and aggregates them by day or session.

## Project Structure

```
codetok/
├── main.go                 # Entrypoint with ldflags version injection
├── cmd/
│   ├── root.go             # Cobra root command
│   ├── daily.go            # codetok daily
│   └── session.go          # codetok session
├── provider/
│   ├── provider.go         # Provider interface and data types
│   └── kimi/
│       └── parser.go       # Kimi CLI wire.jsonl parser
├── stats/
│   └── aggregator.go       # Daily aggregation and date filtering
├── Makefile                # Build, test, lint targets
└── .github/workflows/      # CI and release workflows
```

## Development

```bash
# Build
make build

# Run tests
make test

# Lint (requires golangci-lint)
make lint

# Format code
make fmt

# Tidy dependencies
make tidy

# Show all targets
make help
```

## Adding a New Provider

1. Create a new package under `provider/` (e.g., `provider/claude/`)
2. Implement the `provider.Provider` interface:

```go
type Provider interface {
    Name() string
    CollectSessions(baseDir string) ([]SessionInfo, error)
}
```

3. Wire it into the CLI commands in `cmd/daily.go` and `cmd/session.go`

## License

[MIT](LICENSE)
