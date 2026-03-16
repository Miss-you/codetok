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
- **Cursor** — parses local Cursor usage export CSVs from `~/.codetok/cursor/*.csv`, `~/.codetok/cursor/imports/**/*.csv`, and `~/.codetok/cursor/synced/**/*.csv`

Planned:

- OpenCode

## Installation

### From npm

```bash
npm install -g @yousali/codetok
```

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

Release automation:
- Pushing a `v*` tag publishes GitHub release artifacts.
- The same workflow then publishes the npm package automatically.
- Repository maintainers must configure `NPM_TOKEN` in GitHub Actions secrets.

## Quick Start

```bash
# Show daily token usage dashboard (last 7 days, group-by=cli/provider, unit=m by default)
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

# Read Cursor usage exports from a custom local directory only
codetok daily --all --cursor-dir ~/Downloads/cursor-usage

# Switch aggregation to model view (explicit opt-in)
codetok daily --group-by model

# Show Top 10 groups in the share section
codetok daily --top 10
```

Tip: if you changed code and run `./bin/codetok`, run `make build` first to refresh the binary.

Cursor reporting is local-only:
- `daily` and `session` read local CSV files only.
- They never trigger implicit Cursor login or sync.
- `--cursor-dir` is authoritative and scans only the directory you provide.

## Validation Workflow

Use the documented repository checks before opening a PR:

```bash
make fmt
make vet
make test
make lint
make build
```

To smoke-test the built binary against bundled fixtures instead of your real session logs:

```bash
EMPTY_DIR="$(mktemp -d)"
./bin/codetok daily --all \
  --kimi-dir "$(pwd)/e2e/testdata/sessions" \
  --claude-dir "$EMPTY_DIR" \
  --codex-dir "$EMPTY_DIR" \
  --cursor-dir "$EMPTY_DIR"
rm -rf "$EMPTY_DIR"
```

That command should print the default dashboard sections: `Daily Total Trend`, `CLI Total Ranking`, and `Top 5 CLI Share`.

## Usage

### `codetok daily`

Show daily token usage dashboard.
By default, it shows the last 7 days, grouped by CLI/provider (`--group-by cli`).
Use `--group-by model` to switch to model aggregation.
Use `--all` for full history, or use `--since`/`--until` for an explicit date range.
Dashboard output displays token columns in `m` by default (`--unit m`).
Use `--unit raw`/`k`/`m`/`g` to control display scale.
JSON output always keeps raw integer token counts.
In JSON output, `provider` always keeps provider meaning; grouped dimension and value are described by `group_by` + `group`.
Use `--top N` to control how many groups appear in the share section for the current grouping dimension.
Cursor usage is still local-only in this command: by default `codetok` scans legacy root CSVs plus `imports/` and `synced/` under `~/.codetok/cursor/`. It does not trigger implicit sync.

```
Daily Total Trend
Date   02-15   02-16   02-17   ...
Total  20.32m  8.47m   66.43m  ...
Bar    ###...  #.....  ######  ...

Model/CLI Total Ranking
Rank  Model/CLI        Sessions  Total(m)
1     claude-opus-4-6  23        102.12m
2     gpt-5.3-codex    31        100.83m
3     kimi-for-coding  41        26.78m

Top 5 Model/CLI Share
Rank  Model/CLI        Share   Sessions  Total(m)  Input(m)  Output(m)  Cache Read(m)  Cache Create(m)
1     claude-opus-4-6  43.81%  23        102.12m   0.02m     0.50m      98.21m         3.39m
```

Flags:

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--days` | Lookback window in days when `--since`/`--until` are not set (default: `7`) |
| `--all` | Include all historical sessions (cannot be used with `--days`, `--since`, `--until`) |
| `--unit` | Token display unit for dashboard output: `raw`, `k`, `m`, `g` (default: `m`) |
| `--group-by` | Aggregation dimension for `daily`: `cli` (default, provider/CLI view) or `model` (explicit opt-in) |
| `--top` | Number of groups shown in the share section for the current grouping dimension (default: `5`) |
| `--since` | Start date filter (format: `2006-01-02`) |
| `--until` | End date filter (format: `2006-01-02`) |
| `--provider` | Filter by provider name (e.g. `kimi`, `claude`, `codex`) |
| `--base-dir` | Override default data directory (applies to all providers) |
| `--kimi-dir` | Override Kimi CLI data directory |
| `--claude-dir` | Override Claude Code data directory |
| `--codex-dir` | Override Codex CLI data directory |
| `--cursor-dir` | Override Cursor CSV directory; scans only the provided local path |

Common combinations:
- `codetok daily` — last 7 days, dashboard grouped by CLI/provider, unit `m`
- `codetok daily --unit raw` — last 7 days, raw integer token counts
- `codetok daily --days 30 --unit m` — last 30 days, displayed in millions
- `codetok daily --all --unit g` — full history, displayed in billions
- `codetok daily --group-by model` — switch to model aggregation (explicit opt-in)
- `codetok daily --top 10` — show Top 10 groups in share section

### `codetok session`

Show per-session token usage.
Cursor usage is still local-only in this command: by default `codetok` scans legacy root CSVs plus `imports/` and `synced/` under `~/.codetok/cursor/`. It does not trigger implicit sync.

```
Date        Provider  Session                               Title                      Input     Output  Total
2026-02-13  kimi      75c64dba-5c10-4717-83cd-f3d33abc39bc  Translate article...       72405     6080    78485
2026-02-15  claude    01f3c3c6-a4df-4e2b-8249-ea045ab13f11  Write documentation...     381667    28258   409925
TOTAL                                                                                  2965044   369854  27973571
```

Flags: `--json`, `--since`, `--until`, `--provider`, `--base-dir`, `--kimi-dir`, `--claude-dir`, `--codex-dir`, `--cursor-dir`.
When `--cursor-dir` is set, only that local directory is scanned.

### `codetok version`

Print version information. Commit hash and build date are shown when available.

## How It Works

codetok reads local session data and usage exports stored on disk. Each provider has its own parser that understands the tool's data format. JSONL session files are parsed in parallel using bounded goroutines (default: `min(NumCPU, 8)`, configurable via `CODETOK_WORKERS` env var); Cursor CSV files are discovered from local directories and parsed one file at a time.

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

**Cursor** — `~/.codetok/cursor/*.csv`, `~/.codetok/cursor/imports/**/*.csv`, `~/.codetok/cursor/synced/**/*.csv`
- Parses local Cursor dashboard usage export CSV rows from disk
- Default reporting merges legacy flat files with imported and synced cache CSVs
- `daily` and `session` do not trigger implicit Cursor sync or remote API access
- Maps `Input (w/o Cache Write)`, `Input (w/ Cache Write)`, `Cache Read`, and `Output Tokens` into `codetok` token fields
- Treats each CSV row as one local usage record for session/day views
- Cursor Tab token usage is not supported because the exported data does not provide a defensible Tab token split

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
│   ├── cursor/
│   │   └── parser.go       # Cursor usage CSV parser
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
