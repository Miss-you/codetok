# EBTA-004 Test Strategy

## Red Tests

Add focused tests in `provider/claude/parser_test.go` before production changes:

1. `TestParseClaudeUsageEvents_AssistantUsageEventsUseAssistantTimestamps`
   - One user message followed by assistant usage.
   - Assert one `UsageEvent` using the assistant timestamp, not the user timestamp.
   - Assert provider, session, model, title, workdir hash, source path, event ID, and token fields.

2. `TestParseClaudeUsageEvents_CrossDayAssistantMessages`
   - One session with assistant usage on `2026-04-15` and `2026-04-16`.
   - Assert two events with the same session ID and distinct timestamps.

3. `TestParseClaudeUsageEvents_DeduplicatesStreamingByMessageAndRequest`
   - Reuse the existing streaming shape: same `message.id` and `requestId` with increasing usage.
   - Assert only the final usage/timestamp is emitted for that key.

4. `TestCollectClaudeUsageEvents_IncludesSubagentPaths`
   - Build the same directory shape used by session discovery tests.
   - Assert parent and subagent event files are both collected.

## Verification Commands

Run in this order:

1. Red check:
   - `go test ./provider/claude -run TestParseClaudeUsageEvents`
   - Expected to fail before implementation because `parseUsageEvents` does not exist.

2. Green focused checks:
   - `go test ./provider/claude -run 'Test(ParseClaudeUsageEvents|CollectClaudeUsageEvents)'`
   - `go test ./provider/claude`

3. Repository gates after final scope:
   - `make fmt`
   - `make test`
   - `make vet`
   - `make build`
   - `make lint` if `golangci-lint` is installed

## Coverage Rationale

The focused tests prove the core EBTA-004 contract: native Claude events are timestamped by assistant usage entries, cross-day usage is not collapsed into one session start day, streaming duplicates do not overcount, and subagent discovery stays compatible with existing session collection.
