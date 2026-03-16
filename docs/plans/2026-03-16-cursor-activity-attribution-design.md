# Cursor Activity Attribution Design

## Goal

Add a dedicated Cursor activity attribution capability that reads local Cursor SQLite tracking data and reports `composer` and `tab` line activity without changing any token-report behavior.

## Chosen Approach

Add a new `codetok cursor activity` subcommand backed by a dedicated reader in the top-level `cursor` package. The reader will open `~/.cursor/ai-tracking/ai-code-tracking.db` by default, query Cursor's `scored_commits` table, and aggregate `composerLinesAdded`, `composerLinesDeleted`, `tabLinesAdded`, and `tabLinesDeleted` into a separate activity model.

This keeps activity attribution outside `provider.TokenUsage`, `daily`, and `session`, which preserves the existing token-accounting contract. The command will support both human-readable table output and `--json` output, plus a `--db-path` override for tests and manual inspection.

## Data Model

Use a separate activity result model:

- `db_path`: the database path used by the reader
- `has_data`: whether readable activity rows were found
- `scored_commits`: number of `scored_commits` rows considered
- `composer`: `lines_added`, `lines_deleted`
- `tab`: `lines_added`, `lines_deleted`

The model intentionally avoids token field names and does not embed `provider.TokenUsage`.

## Reader Behavior

- Default path: `~/.cursor/ai-tracking/ai-code-tracking.db`
- Missing database: return a no-data result, not an error
- Unreadable database: return a no-data result, not an error
- Missing `scored_commits` table: return a no-data result, not an error
- Valid database with only one category populated: return the present category and zero values for the other

## CLI Behavior

`codetok cursor activity`

- Default output: table labeled as activity attribution
- `--json`: emit the activity model as JSON
- `--db-path`: override the default tracking database path

The command text must consistently use `activity` / `attribution` wording, not token wording.

## Testing

- Unit tests for the SQLite reader with generated SQLite fixtures:
  - database exists with both categories
  - database missing
  - database with composer only
  - database with tab only
- Command tests for `cursor activity` JSON and table output
- End-to-end checks confirming `daily` and `session` JSON output remain unchanged even when activity data exists separately

## Non-Goals

- Do not merge activity attribution into `provider.TokenUsage`
- Do not add activity fields to `daily` or `session`
- Do not infer Cursor token totals from activity attribution
