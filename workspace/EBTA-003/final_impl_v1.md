# EBTA-003 Final Implementation v1

## Scope

Implement native Codex usage events only. Do not switch `daily` or `session` to event aggregation in this task; later EBTA tasks own command integration.

Touched production surface:

- `provider/codex/parser.go`
- `provider/parallel.go`

Touched tests:

- `provider/codex/parser_test.go`

## Implementation

Add `(*Provider).CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)` beside `CollectSessions`.

Refactor Codex session-file discovery into shared helpers:

- explicit `baseDir` wins unchanged
- otherwise `$CODEX_HOME/sessions` wins when `CODEX_HOME` is set
- otherwise fall back to `~/.codex/sessions`
- collect the same `baseDir/<year>/<month>/<day>/*.jsonl` files for sessions and events
- parse usage event files with bounded parallelism through a shared provider helper

Add `parseCodexUsageEvents(path string)`:

- scan one JSONL rollout file
- track stable session metadata from the first `session_meta.id`
- track title from the first `user_message`
- track the current valid model from `turn_context`, `event_msg.model`, `event_msg.info`, and existing raw JSON model paths
- emit one `provider.UsageEvent` per non-zero token delta
- set `ProviderName=codex`, `SessionID`, `Title`, `ModelName`, `Timestamp`, `TokenUsage`, `SourcePath`, and a deterministic file-local `EventID`

Token rules:

- prefer `info.last_token_usage` when present
- otherwise use `info.total_token_usage` minus the previous cumulative total in the same file
- update the previous cumulative total whenever `total_token_usage` is present
- if any cumulative counter decreases, treat it as a counter reset: emit the current total as the fresh delta and replace the baseline
- map `input_tokens` and `cached_input_tokens` the same way as the existing session parser: `InputOther=input_tokens-cached_input_tokens`, `InputCacheRead=cached_input_tokens`
- keep `Output=output_tokens`; do not add `reasoning_output_tokens` separately in this change
- skip malformed JSON, null `info`, invalid timestamps, and zero deltas

Keep `parseCodexSession` behavior compatible with existing tests. Sharing the base-dir resolver means `CollectSessions("")` also honors `CODEX_HOME`, which is required by the task.

## Boundaries

No changes to:

- `cmd/`
- `stats/`
- other providers
- README/help text

No remote calls are introduced.
