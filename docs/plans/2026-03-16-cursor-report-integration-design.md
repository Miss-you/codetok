# Cursor Report Integration Design

**Date:** 2026-03-16

## Scope

Implement OpenSpec change `cursor-usage-support` for:
- `tasks.md` 3.1 through 3.5
- `tasks.md` 5.2 and 5.3

This ticket only changes local token reporting behavior. It does not implement Cursor activity attribution or any remote sync/login workflow.

## Approved Direction

Use a narrow default Cursor root layout:
- root-level legacy CSV files under `~/.codetok/cursor/*.csv`
- imported CSV files under `~/.codetok/cursor/imports/**/*.csv`
- synced CSV cache files under `~/.codetok/cursor/synced/**/*.csv`

When `--cursor-dir` is provided, that directory becomes authoritative and is scanned by itself. Reporting must not merge any default Cursor paths, inspect local Cursor credentials, or trigger network behavior.

## Architecture

Add a shared session collection helper in `cmd/` and route both `daily` and `session` through it. That keeps provider filtering, per-provider directory override precedence, missing-directory handling, and error behavior identical between the two commands.

Keep the provider interface unchanged. `provider/cursor` remains a local file reader that resolves CSV paths and parses them into `provider.SessionInfo`.

## Behavior

- Default reporting merges imported and synced Cursor CSV files.
- Legacy flat CSV files at the Cursor root stay supported.
- Missing Cursor default directories are skipped without breaking other providers.
- Malformed Cursor CSV files are skipped when other valid local CSV files exist.
- `daily` and `session` continue to be local-only reporting commands.

## Testing

- Unit tests for the shared command collection helper.
- Provider tests for default Cursor root discovery and explicit override behavior.
- E2E tests for:
  - import-only
  - sync-only
  - import + sync + legacy coexistence
  - invalid sync file with valid cached local data still reporting
  - explicit `--cursor-dir` override
  - no implicit remote access for `daily` and `session`
