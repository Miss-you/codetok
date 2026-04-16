# EBTA-003 Original Codex Implementation

## Current behavior

The Codex provider is still session-based. `provider/codex/parser.go` exposes only `CollectSessions(baseDir string)`, which walks a Codex sessions tree, parses each `rollout-*.jsonl` file, and returns one `provider.SessionInfo` per file.

Inside each file, the parser:

- reads `session_meta` to capture `SessionID` and an optional start timestamp,
- reads `event_msg` records to count `user_message` turns,
- reads `token_count` records and keeps only the latest cumulative token totals it sees,
- extracts `ModelName` from several JSON paths when present,
- sets `StartTime` and `EndTime` from the timestamps it encounters while scanning the file,
- skips malformed lines and malformed payloads instead of failing the whole session.

There is no native `CollectUsageEvents` implementation yet, and Codex does not emit timestamped token deltas. All reporting still depends on collapsing a whole file into a single `SessionInfo`.

## Test coverage today

`provider/codex/parser_test.go` covers:

- a valid Codex session file,
- empty files,
- malformed JSON lines,
- sessions with no token_count records,
- model extraction from `token_count` payloads and other payload shapes,
- placeholder model rejection,
- multiple `token_count` records, where the last cumulative record wins,
- directory traversal across `year/month/day` folders,
- basic provider metadata such as `ProviderName`, `SessionID`, `Title`, `Turns`, and token totals.

What the tests do not cover yet:

- event-level timestamps,
- per-token deltas,
- native usage-event collection,
- `CODEX_HOME` resolution,
- any cross-day attribution behavior.

## Source directory resolution

`CollectSessions` currently uses this fallback only when `baseDir` is empty:

- `os.UserHomeDir()` + `/.codex/sessions`

It does not check `CODEX_HOME`, so the active source directory is effectively hard-coded to the home-directory default unless the caller passes an explicit path.

The directory scan is also fixed to the Codex rollout layout:

- `baseDir/<year>/<month>/<day>/rollout-*.jsonl`

## What is missing for native UsageEvent support

To support the event-first aggregation plan, Codex still needs to:

- emit one `provider.UsageEvent` per timestamped token record,
- preserve the event timestamp instead of only session start/end bounds,
- prefer `last_token_usage` when present,
- compute deltas when only cumulative `total_token_usage` values are available,
- keep model/session metadata attached to each event,
- honor `CODEX_HOME` before falling back to `~/.codex/sessions`,
- expose a native `CollectUsageEvents` path so stats can aggregate by event date rather than by session start date.

