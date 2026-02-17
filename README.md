# codetok

[![CI](https://github.com/Miss-you/codetok/actions/workflows/ci.yml/badge.svg)](https://github.com/Miss-you/codetok/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)

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
# Show daily token usage breakdown (all providers)
codetok daily

# Show per-session token usage
codetok session

# Output as JSON
codetok daily --json

# Filter by date range
codetok daily --since 2026-02-01 --until 2026-02-15

# Filter by provider
codetok daily --provider claude
codetok session --provider kimi
```

## Usage

### `codetok daily`

Show daily token usage breakdown.

```
Date        Provider  Sessions  Input    Output  Cache Read  Cache Create  Total
2026-02-07  kimi      5         109822   15356   632985      0             758163
2026-02-08  claude    2         95046    7010    274232      0             376288
2026-02-15  codex     21        938566   149287  7869696     0             8957549
TOTAL                 49        2965044  369854  24638673    0             27973571
```

Flags:

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--since` | Start date filter (format: `2006-01-02`) |
| `--until` | End date filter (format: `2006-01-02`) |
| `--provider` | Filter by provider name (e.g. `kimi`, `claude`, `codex`) |
| `--base-dir` | Override default data directory (applies to all providers) |
| `--kimi-dir` | Override Kimi CLI data directory |
| `--claude-dir` | Override Claude Code data directory |
| `--codex-dir` | Override Codex CLI data directory |

### `codetok session`

Show per-session token usage.

```
Date        Provider  Session                               Title                      Input     Output  Total
2026-02-13  kimi      75c64dba-5c10-4717-83cd-f3d33abc39bc  Translate article...       72405     6080    78485
2026-02-15  claude    01f3c3c6-a4df-4e2b-8249-ea045ab13f11  Write documentation...     381667    28258   409925
TOTAL                                                                                  2965044   369854  27973571
```

Flags: same as `codetok daily`.

### `codetok version`

Print build version, commit hash, and build date.

## How It Works

codetok reads local session data that AI coding CLIs store on disk. Each provider has its own parser that understands the tool's data format. All session files are parsed in parallel using bounded goroutines (default: `min(NumCPU, 8)`, configurable via `CODETOK_WORKERS` env var).

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

import "github.com/Miss-you/codetok/provider"

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
   _ "github.com/Miss-you/codetok/provider/myprovider"
   ```
4. Add `--myprovider-dir` flag if needed

## License

[MIT](LICENSE)
