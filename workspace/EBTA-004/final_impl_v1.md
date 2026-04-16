# EBTA-004 Final Implementation v1

## Scope

Implement native Claude `provider.UsageEvent` collection without changing the existing `CollectSessions` contract or command behavior. This task is scoped to `provider/claude/parser.go` and `provider/claude/parser_test.go`.

OpenSpec handling: this task uses the existing active `event-based-token-aggregation-core` change created by EBTA-001. That change already defines the shared `UsageEvent` and optional `UsageEventProvider` API. EBTA-004 is a provider implementation task under the approved board, so no additional OpenSpec change is needed.

## Current Behavior

`CollectSessions` discovers Claude JSONL files under explicit `baseDir`, or under `~/.claude/projects` and `~/.claude-internal/projects` by default. Discovery includes top-level session JSONL files and nested `<session>/subagents/*.jsonl`.

`parseSession` currently collapses all assistant `message.usage` entries into one `SessionInfo.TokenUsage`. It already deduplicates streaming entries by `message.id + requestId`, with unique fallback keys when both IDs are missing.

## Target Behavior

Add:

- `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)`
- `func parseUsageEvents(path, projectSlug string) ([]provider.UsageEvent, error)`

Collection reuses the same path discovery semantics as `CollectSessions`.

Parsing emits one usage event per deduplicated assistant usage entry:

- `ProviderName`: `claude`
- `SessionID`: event `sessionId`, with filename fallback after parsing
- `ModelName`: assistant `message.model` when present
- `Title`: first external user message text, truncated the same way as sessions
- `WorkDirHash`: project slug from path discovery
- `Timestamp`: assistant event timestamp
- `TokenUsage`: same field mapping as existing session aggregation
- `SourcePath`: source JSONL path
- `EventID`: dedupe key

Streaming dedupe keeps the latest assistant usage and timestamp for a stable `message.id + requestId` key. Entries missing both IDs remain unique.

## Integration Notes

Refactor path discovery into a small shared helper inside `provider/claude/parser.go` so `CollectSessions` and `CollectUsageEvents` keep identical explicit/default directory behavior.

Do not change:

- token field semantics
- malformed-line skip behavior
- session parser totals
- subagent discovery rules
- default/explicit directory error handling

## Review Criteria

- Native events preserve assistant timestamps instead of session start time.
- Cross-day assistant messages from the same session remain separate events.
- Streaming duplicates keep only the latest usage.
- Subagent files are included through the same discovery path as sessions.
- Existing Claude session tests remain green.
