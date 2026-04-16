## Why

Codex rollout files contain timestamped `token_count` records, but the provider currently collapses them into one session summary. Native usage events are needed so later command tasks can attribute Codex usage to the actual token event date instead of the session start date.

## What Changes

- Add native `CollectUsageEvents` support to the Codex provider.
- Parse Codex `token_count` records into timestamped `provider.UsageEvent` deltas.
- Prefer `last_token_usage` when present and derive deltas from cumulative `total_token_usage` otherwise.
- Preserve stable Codex session, title, model, source path, and event timestamp metadata on emitted events.
- Resolve empty Codex provider roots from `$CODEX_HOME/sessions` before falling back to `~/.codex/sessions`.
- No breaking CLI behavior changes in this task.

## Capabilities

### New Capabilities

- `codex-native-usage-events`: Codex provider support for native timestamped usage events from local rollout JSONL files.

### Modified Capabilities

- None.

## Impact

- Affected code: `provider/codex/parser.go`, `provider/codex/parser_test.go`, `provider/parallel.go`.
- No new dependencies.
- No remote API calls.
- Command integration remains owned by later event-based aggregation tasks.
