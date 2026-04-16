# EBTA-004 Claude Native Usage Events Plan

## Goal

Add Claude Code native `provider.UsageEvent` collection from local JSONL assistant
`message.usage` records, without changing existing `CollectSessions` behavior.

## Files

- Modify `provider/claude/parser.go`.
- Modify `provider/claude/parser_test.go`.
- Do not modify `provider/provider.go` or `stats/events.go`; EBTA-001 already added
  the shared event model and aggregation helpers.

## New Function Signatures

Add these in `provider/claude/parser.go`:

```go
func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)

func parseUsageEvents(path, projectSlug string) ([]provider.UsageEvent, error)
```

Optional small private helpers are acceptable if they keep behavior identical:

```go
func usageFromClaudeUsage(u *claudeUsage) provider.TokenUsage

func applyClaudeEventMetadata(events []provider.UsageEvent, sessionID, title, modelName string) []provider.UsageEvent
```

Keep the existing `dedupKey(messageID, requestID string, counter *int) string`
helper and reuse it for event dedupe so session and event parsing use the same
identity rule.

## Collection Flow

`CollectUsageEvents` should mirror `CollectSessions` path discovery:

1. Use explicit `baseDir` when provided.
2. When `baseDir` is empty, scan both `~/.claude/projects` and
   `~/.claude-internal/projects`.
3. Use existing `collectPaths` so top-level session files and
   `<session-uuid>/subagents/*.jsonl` files are included.
4. Preserve current error semantics:
   - explicit missing/unreadable directory returns an error;
   - default missing directories are skipped;
   - default non-missing operational errors are returned.
5. Parse discovered files with `provider.ParseParallel(paths, 0, ...)`, returning
   `[]provider.UsageEvent` per path and flattening the result.

If `provider.ParseParallel` cannot directly flatten slices, add a local loop
around the returned `[][]provider.UsageEvent`. Do not add a new dependency.

## Event Parsing Semantics

`parseUsageEvents(path, projectSlug)` should scan one JSONL file with the same
scanner buffer size and malformed-line behavior as `parseSession`.

Only assistant events with non-nil `message.usage` emit usage events.

Each emitted event should set:

- `ProviderName: "claude"`
- `SessionID`: prefer the assistant event's `sessionId`; after the scan, fill
  blank events with the first session id seen in the file; if still blank, use
  the filename without `.jsonl`, matching `parseSession`.
- `ModelName`: prefer the assistant event's `message.model`; after the scan,
  fill blank events with the first non-empty assistant model seen in the file.
  Do not overwrite per-event model values.
- `Title`: first external user message text via existing `extractUserText`,
  truncated with `truncateTitle(title, 80)`, applied to all events after scan.
- `WorkDirHash: projectSlug`
- `Timestamp`: the parsed assistant event timestamp.
- `TokenUsage`: direct mapping from Claude usage fields:
  `input_tokens -> InputOther`,
  `cache_creation_input_tokens -> InputCacheCreate`,
  `cache_read_input_tokens -> InputCacheRead`,
  `output_tokens -> Output`.
- `SourcePath: path`
- `EventID`: the stable dedupe key for records with at least one id, and a
  source-line fallback for records with neither id.

Do not sum usage inside `parseUsageEvents`; each deduped assistant usage record
is one timestamped event.

Skip malformed JSON lines as today. For assistant usage lines with an
unparseable or empty timestamp, skip the usage event instead of fabricating a
date; native event aggregation depends on event timestamps.

## Streaming Dedupe Algorithm

Claude streaming can write the same assistant message multiple times with
increasing usage. Dedupe by the composite `message.id + requestId` key and keep
the latest usage for that key.

Concrete algorithm:

```go
type claudeUsageEventEntry struct {
	event     provider.UsageEvent
	timestamp time.Time
	sequence  int
}

deduped := map[string]claudeUsageEventEntry{}
uniqueCounter := 0
sequence := 0

// for each assistant event with usage and a valid timestamp:
sequence++
key := dedupKey(event.Message.ID, event.RequestID, &uniqueCounter)
candidate := claudeUsageEventEntry{
	event: provider.UsageEvent{...},
	timestamp: parsedAssistantTimestamp,
	sequence: sequence,
}

previous, exists := deduped[key]
if !exists ||
	candidate.timestamp.After(previous.timestamp) ||
	(candidate.timestamp.Equal(previous.timestamp) && candidate.sequence > previous.sequence) {
	deduped[key] = candidate
}
```

For records where both `message.id` and `requestId` are empty, `dedupKey` already
returns a unique key, so those usage records are deliberately not merged.

After scanning:

1. Convert map values into a slice.
2. Enrich blank `SessionID`, `ModelName`, and `Title` with file-level metadata.
3. Sort events deterministically by `Timestamp`, then `SourcePath`, then
   `EventID`.

Sorting keeps tests stable and makes native event output deterministic without
affecting token totals.

## Metadata Preservation

Project and source metadata should come from the same discovery path used by
session collection:

- `collectPaths` already records `pathToSlug[path] = projectSlug`.
- Pass that slug into `parseUsageEvents(path, projectSlug)`.
- Set every event's `WorkDirHash` to `projectSlug`.
- Set every event's `SourcePath` to the exact scanned JSONL path. This is
  important because `stats.eventSessionKey` falls back to `SourcePath` when
  `SessionID` is absent.
- Subagent files keep the parent project slug because `collectPaths` assigns the
  parent project directory name to subagent paths today.

Session metadata should stay file-local:

- `SessionID` comes from JSONL events where present, then filename fallback.
- `Title` comes from the first external user message in that same JSONL file.
- `ModelName` is per assistant event first, file fallback second.

## Test Structure

Add focused tests in `provider/claude/parser_test.go`. Keep them table-light and
use inline temp JSONL files so expected timestamps and token totals are obvious.

1. `TestParseClaudeUsageEvents_CrossDayAssistantMessages`
   - One file with a first user message before midnight and two assistant usage
     messages on different dates.
   - Assert two events are returned.
   - Assert each event timestamp equals its assistant line timestamp, not the
     first user timestamp or session start.
   - Assert both events keep the same `SessionID`, `WorkDirHash`, `SourcePath`,
     and title.

2. `TestParseClaudeUsageEvents_UsesAssistantTimestamp`
   - Minimal file where user timestamp and assistant timestamp are on different
     days.
   - Assert the single event timestamp is exactly the assistant timestamp.
   - This protects EBTA-004 from accidentally deriving event dates from
     `SessionInfo.StartTime`.

3. `TestParseClaudeUsageEvents_DedupStreamingKeepsLatestUsage`
   - Three assistant usage lines share the same `message.id` and `requestId`
     with increasing output tokens.
   - Add a second assistant usage line with a different key.
   - Assert only two events remain.
   - Assert the shared-key event keeps the latest token fields and latest
     timestamp.

4. `TestParseClaudeUsageEvents_NoIDsRemainUnique`
   - Two assistant usage lines without `message.id` and without `requestId`.
   - Assert both events remain, because there is no stable streaming key.

5. `TestCollectClaudeUsageEvents_IncludesSubagents`
   - Build a temp Claude project with:
     - `project-x/session-abc.jsonl`
     - `project-x/session-abc/subagents/agent-a123.jsonl`
   - Call `(&Provider{}).CollectUsageEvents(baseDir)`.
   - Assert events from both source paths are present.
   - Assert both events have `WorkDirHash == "project-x"`.

Targeted gate:

```bash
go test ./provider/claude -run 'Test(ParseClaudeUsageEvents|CollectClaudeUsageEvents)'
```

Broader gate after implementation:

```bash
go test ./provider/claude
```

## What Stays Unchanged In CollectSessions

Do not route `CollectSessions` through the new event parser in EBTA-004.

Leave these behaviors intact:

- `func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error)`
  remains the provider session API implementation.
- Existing session token aggregation remains session-level and summed into one
  `SessionInfo.TokenUsage`.
- Existing session streaming dedupe behavior remains unchanged.
- Existing user-turn counting, title extraction, start/end time computation,
  model fallback, and filename session-id fallback remain unchanged.
- Existing `collectPaths` semantics remain unchanged, including subagent
  discovery and explicit/default directory error behavior.
- `CollectSessions` remains local-only and must not call Claude remote APIs.
- Existing `TestParseClaudeSession_*` and `TestCollectClaudeSessions_*` tests
  should continue passing without assertion changes.

EBTA-004 should add native event collection beside session collection; command
integration and any switch from session aggregation to event aggregation belongs
to the collector bridge task, not this provider task.
