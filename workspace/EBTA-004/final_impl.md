# EBTA-004 Final Implementation

## Decision

Add Claude native `provider.UsageEvent` collection additively:

- `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)`
- `func parseUsageEvents(path, projectSlug string) ([]provider.UsageEvent, error)`

This task uses the existing active OpenSpec change `event-based-token-aggregation-core`. No new OpenSpec change is created because EBTA-001 already introduced the shared event API and EBTA-004 is the Claude provider implementation slice from the approved task board.

## Collection

`CollectUsageEvents` will reuse the existing `collectPaths` discovery behavior so explicit/default directories and subagent inclusion match `CollectSessions` exactly:

- explicit `baseDir`: scan only that directory and return read errors
- default `baseDir`: scan `~/.claude/projects` and `~/.claude-internal/projects`, ignore missing defaults, return non-missing operational errors
- include top-level `*.jsonl` and `<session>/subagents/*.jsonl`

`provider.ParseParallel` is intentionally not used for events in EBTA-004 because its callback type returns `provider.SessionInfo`. Forcing event slices through it would add avoidable adapter complexity. This task keeps event collection simple and local to the Claude provider.

## Parsing

`parseUsageEvents` scans a single JSONL file with the same malformed-line tolerance and scanner buffer size as `parseSession`.

Only assistant events with non-nil `message.usage` and a valid RFC3339 timestamp produce events. Invalid usage timestamps are skipped instead of producing zero-time events.

Each deduplicated event sets:

- `ProviderName`: `claude`
- `SessionID`: assistant `sessionId`, then file-level first session ID, then filename stem
- `ModelName`: per-assistant `message.model`, then file-level first assistant model
- `Title`: first accepted external user text, truncated to 80 runes and filled after scan
- `WorkDirHash`: project slug from discovery
- `Timestamp`: assistant event timestamp
- `TokenUsage`: same Claude field mapping as session parsing
- `SourcePath`: scanned JSONL path
- `EventID`: dedupe key

## Dedupe

Streaming rows are deduplicated with the existing `dedupKey(message.id, requestId, &uniqueCounter)` identity rule.

For a stable key, the latest row in file order wins, matching existing `parseSession` behavior. Rows with neither ID keep generated unique keys and are not merged.

Output events are sorted deterministically by timestamp, source path, and event ID after metadata enrichment. Sorting only stabilizes native event output; it does not change token totals.

## Tests

Add failing tests first for:

- assistant usage events using assistant timestamps
- cross-day assistant usage from one session
- streaming dedupe keeping latest usage/timestamp
- no-ID assistant usage records remaining unique
- `CollectUsageEvents` including subagent JSONL paths

Then implement the minimal parser/collector needed to pass them and run the package/full repository gates.

## Non-Changes

Do not change:

- `CollectSessions`
- session token aggregation
- user turn counting
- title extraction rules
- session start/end time computation
- token field semantics
- malformed JSON skip behavior
- local-only provider behavior
