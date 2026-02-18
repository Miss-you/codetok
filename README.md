# codetok

[![CI](https://github.com/miss-you/codetok/actions/workflows/ci.yml/badge.svg)](https://github.com/miss-you/codetok/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)

[中文文档](README_zh.md)

A CLI tool for tracking and aggregating token usage across AI coding CLI tools.

Supported providers:

- **Kimi CLI** — parses `~/.kimi/sessions/**/wire.jsonl`
- **Claude Code** — parses `~/.claude/projects/**/*.jsonl` (with streaming deduplication)
- **Codex CLI** — parses `~/.codex/sessions/**/*.jsonl`

Planned:

- OpenCode
- Cursor

## Installation

### From source

```bash
go install github.com/miss-you/codetok@latest
```

Note: Go module paths are case-sensitive. Use `github.com/miss-you/codetok` exactly (all lowercase).

### Build locally

```bash
git clone https://github.com/miss-you/codetok.git
cd codetok
make build
# Binary at ./bin/codetok
```

### From release

Download pre-built binaries from the [Releases](https://github.com/miss-you/codetok/releases) page.

## Quick Start

```bash
# Show daily token usage breakdown (last 7 days, unit=k by default)
codetok daily

# Show per-session token usage
codetok session

# Output as JSON
codetok daily --json

# Show all historical daily usage
codetok daily --all

# Use a custom rolling window
codetok daily --days 30

# Show raw integers instead of unit-scaled values
codetok daily --unit raw

# Force display in millions
codetok daily --unit m

# Filter by date range
codetok daily --since 2026-02-01 --until 2026-02-15

# Filter by provider
codetok daily --provider claude
codetok session --provider kimi
```

Tip: if you changed code and run `./bin/codetok`, run `make build` first to refresh the binary.

## Usage

### `codetok daily`

Show daily token usage breakdown.
By default, it shows the last 7 days.
Use `--all` for full history, or use `--since`/`--until` for an explicit date range.
Table output displays token columns in `k` by default (`--unit k`).
Use `--unit raw`/`k`/`m`/`g` to control display scale.
JSON output always keeps raw integer token counts.

```
Date        Provider  Sessions  Input(k)  Output(k)  Cache Read(k)  Cache Create(k)  Total(k)
2026-02-07  kimi      5         109.82k  15.36k  632.98k     0.00k         758.16k
2026-02-08  claude    2         95.05k   7.01k   274.23k     0.00k         376.29k
2026-02-15  codex     21        938.57k  149.29k 7869.70k    0.00k         8957.55k
TOTAL                 49        2965.04k 369.85k 24638.67k   0.00k         27973.57k
```

Flags:

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--days` | Lookback window in days when `--since`/`--until` are not set (default: `7`) |
| `--all` | Include all historical sessions (cannot be used with `--days`, `--since`, `--until`) |
| `--unit` | Token display unit for table output: `raw`, `k`, `m`, `g` (default: `k`) |
| `--since` | Start date filter (format: `2006-01-02`) |
| `--until` | End date filter (format: `2006-01-02`) |
| `--provider` | Filter by provider name (e.g. `kimi`, `claude`, `codex`) |
| `--base-dir` | Override default data directory (applies to all providers) |
| `--kimi-dir` | Override Kimi CLI data directory |
| `--claude-dir` | Override Claude Code data directory |
| `--codex-dir` | Override Codex CLI data directory |

Common combinations:
- `codetok daily` — last 7 days, table unit `k`
- `codetok daily --unit raw` — last 7 days, raw integer token counts
- `codetok daily --days 30 --unit m` — last 30 days, displayed in millions
- `codetok daily --all --unit g` — full history, displayed in billions

### `codetok session`

Show per-session token usage.

```
Date        Provider  Session                               Title                      Input     Output  Total
2026-02-13  kimi      75c64dba-5c10-4717-83cd-f3d33abc39bc  Translate article...       72405     6080    78485
2026-02-15  claude    01f3c3c6-a4df-4e2b-8249-ea045ab13f11  Write documentation...     381667    28258   409925
TOTAL                                                                                  2965044   369854  27973571
```

Flags: `--json`, `--since`, `--until`, `--provider`, `--base-dir`, `--kimi-dir`, `--claude-dir`, `--codex-dir`.

### `codetok version`

Print version information. Commit hash and build date are shown when available.

## How It Works

codetok reads local session data that AI coding CLIs store on disk. Each provider has its own parser that understands the tool's data format. All session files are parsed in parallel using bounded goroutines (default: `min(NumCPU, 8)`, configurable via `CODETOK_WORKERS` env var).

Statistics scope:
- Token usage is computed by aggregating token counters from existing local session logs.
- codetok does not call provider APIs.
- Sessions are counted only if their local log files currently exist.

**Kimi CLI** — `~/.kimi/sessions/<work-dir-hash>/<session-uuid>/wire.jsonl`
- Parses `StatusUpdate` events containing `token_usage`

**Claude Code** — `~/.claude/projects/<project-slug>/<session-uuid>.jsonl`
- Parses `assistant` events with `message.usage`
- Deduplicates streaming events using `messageId:requestId` composite key (last-entry-wins)

**Codex CLI** — `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`
- Parses `event_msg` events with `payload.type="token_count"`
- Takes the last (cumulative) token count per session

## Project Structure

```
codetok/
├── main.go                 # Entrypoint with ldflags version injection
├── cmd/
│   ├── root.go             # Cobra root command
│   ├── daily.go            # codetok daily (multi-provider)
│   └── session.go          # codetok session (multi-provider)
├── provider/
│   ├── provider.go         # Provider interface and data types
│   ├── registry.go         # Provider auto-registration via init()
│   ├── parallel.go         # Bounded parallel parsing helper
│   ├── kimi/
│   │   └── parser.go       # Kimi CLI wire.jsonl parser
│   ├── claude/
│   │   └── parser.go       # Claude Code JSONL parser (with dedup)
│   └── codex/
│       └── parser.go       # Codex CLI JSONL parser
├── stats/
│   └── aggregator.go       # Daily aggregation and date filtering
├── e2e/                    # End-to-end tests
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

1. Create a new package under `provider/` (e.g., `provider/myprovider/`)
2. Implement the `provider.Provider` interface and register via `init()`:

```go
package myprovider

import "github.com/miss-you/codetok/provider"

func init() {
    provider.Register(&Provider{})
}

type Provider struct{}

func (p *Provider) Name() string { return "myprovider" }

func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
    // Parse session files, use provider.ParseParallel for concurrent parsing
    // ...
}
```

3. Import the package in `cmd/daily.go` and `cmd/session.go` with a blank import:
   ```go
   _ "github.com/miss-you/codetok/provider/myprovider"
   ```
4. Add `--myprovider-dir` flag if needed

## License

[MIT](LICENSE)
