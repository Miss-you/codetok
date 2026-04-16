# AGENTS.md

This file provides durable guidance to coding agents working in this repository.

## Scope

- Build `codetok` as a Go CLI for aggregating token usage and local activity data from coding assistant artifacts.
- Statistics commands aggregate counters from files that already exist on disk; they must not silently call provider remote APIs.
- Keep this file concise and stable. Put task-specific plans, investigations, and temporary notes in `docs/plans/`, `openspec/changes/`, or another task artifact.

## Source Of Truth

- Current CLI behavior: command code in `cmd/`, focused tests in `cmd/`, and e2e tests in `e2e/`.
- User-facing behavior: `README.md` and `README_zh.md`.
- Architecture background: `CLAUDE.md`.
- If documentation, chat history, and repository state disagree, trust code plus tests first, then update the stale documentation as part of the change.

## Build And Verification

```bash
make build          # Build binary to ./bin/codetok
make test           # Run all tests with -race -cover
make lint           # Run golangci-lint if installed
make fmt            # Format code with go fmt and goimports when available
make vet            # Run go vet
make tidy           # go mod tidy + go mod verify
```

Always run the narrowest relevant test first, then the broader gate needed for the change. Run `make test` after code changes unless the change is documentation-only.

Before manual CLI verification with `./bin/codetok`, run `make build`; `go run . ...` can show newer behavior than an old binary.

## Command Contracts

- `codetok daily` defaults to a rolling 7-day window.
- `codetok daily` defaults to CLI/provider grouping (`--group-by cli`); model grouping is explicit with `--group-by model`.
- Dashboard output uses `--unit m` by default. `--unit raw|k|m|g` affects dashboard output only; JSON keeps raw token counts.
- `codetok daily --top N` controls the share section size for the active grouping dimension; the default is `5`.
- `codetok daily --all` includes full history and is mutually exclusive with `--days`, `--since`, and `--until`.
- `codetok daily --days N` is mutually exclusive with `--since` and `--until`.
- JSON grouping semantics preserve `provider` as provider identity; the active aggregation dimension is described by `group_by` and `group`.
- `daily`, `session`, and `cursor activity` read local files only and must not trigger implicit login or sync.
- `cursor login`, `cursor status`, and `cursor sync` are the explicit commands that may contact the Cursor API.
- When `--cursor-dir` is set, it is authoritative and scans only that local directory.

## Architecture Rules

- `cmd/` owns Cobra command wiring, flags, terminal output, and JSON response shapes.
- `provider/` owns provider parsers, shared token/session types, the provider registry, and bounded parallel parsing.
- Provider packages self-register through `provider.Register()` in `init()` and are imported by CLI commands with blank imports.
- `stats/` owns date filtering and day-level aggregation. Keep aggregation rules out of provider parsers.
- `cursor/` owns Cursor API/store/activity service behavior. Keep reporting commands local-only unless the user invokes an explicit Cursor network command.
- `e2e/` owns binary-level behavior checks and fixture-driven CLI contracts.
- Keep provider-specific parsing in `provider/<name>/`; keep cross-provider behavior in shared packages only when at least two providers need it.
- Skip malformed local data gracefully where existing parsers do so, but wrap operational errors with context using `fmt.Errorf("context: %w", err)`.

## Provider Rules

When adding or changing a provider:

- Implement the `provider.Provider` interface in `provider/<name>/`.
- Use `provider.ParseParallel()` for concurrent file parsing where multiple files are scanned.
- Add unit fixtures under the provider package `testdata/`.
- Add or update e2e fixtures under `e2e/testdata/` when the CLI contract changes.
- Add blank imports in commands that should include the provider.
- Add a `--<name>-dir` override when users need to point the provider at a custom local data directory.
- Preserve local-only reporting semantics; do not add implicit provider API calls to `daily` or `session`.

## Workflow

- Prefer small, package-scoped changes that preserve existing CLI contracts.
- Update tests and docs in the same change when behavior changes.
- Keep terminal tables built with `text/tabwriter` unless there is a clear reason to change the output stack.
- Use snake_case JSON tags for user-facing JSON fields.
- Keep imports grouped as standard library, external dependencies, then internal packages; rely on `make fmt`.
- Do not broaden dependencies casually. This CLI should stay small and predictable.
