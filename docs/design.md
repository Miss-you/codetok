# Codetok Kimi Code Parser — Design Doc

## Data Source

Kimi Code stores session data at `~/.kimi-code/sessions/<work-dir>/<session-id>/`:
- `state.json`: session metadata (title, createdAt, updatedAt)
- `agents/<agent>/wire.jsonl`: per-agent event stream containing token usage in `usage.record` events
  (`main` is the user-facing agent; `agent-N` are subagents — their usage is summed into the session)

### wire.jsonl Format

Each line is a JSON object with a top-level `type` field and a `time` field in
milliseconds since epoch. Event types:
- `metadata` (line 1): `{"type": "metadata", "protocol_version": "1.5", "created_at": ...}`
- `turn.prompt`: a user prompt begins a turn
- `usage.record`: contains per-request `usage` — **this is what we parse**
- `turn.ended`: marks end of a turn
- Others: `profile.bind`, `context.append_message`, `llm.request`, `llm.tools_snapshot`, ...

### usage.record Event (key data)

```json
{
  "type": "usage.record",
  "model": "kimi-code/k3-256k",
  "usage": {
    "inputOther": 25569,
    "output": 188,
    "inputCacheRead": 0,
    "inputCacheCreation": 0
  },
  "usageScope": "turn",
  "time": 1787806799376
}
```

`usageScope` is `turn` for regular LLM requests and `session` for internal calls
(e.g. compaction); both are summed. The `kimi-code/` model-alias prefix is stripped
for display (`kimi-code/k3-256k` → `k3-256k`).

## Package Structure

```
codetok/
├── main.go                    # entrypoint, ldflags
├── cmd/
│   ├── root.go                # cobra root command
│   ├── daily.go               # `codetok daily` subcommand
│   └── session.go             # `codetok session` subcommand
├── provider/
│   ├── provider.go            # Provider interface + common types
│   └── kimi/
│       ├── parser.go          # Kimi Code wire.jsonl parser
│       ├── parser_test.go     # Unit tests
│       └── testdata/          # Test fixtures
│           ├── wire.jsonl
│           └── state.json
├── stats/
│   ├── aggregator.go          # Aggregate by day/session
│   └── aggregator_test.go     # Unit tests
└── e2e/
    ├── e2e_test.go            # End-to-end CLI tests
    └── testdata/              # Full session fixture tree
        └── sessions/
            └── abc123/
                └── uuid-1/
                    ├── state.json
                    └── agents/
                        └── main/
                            └── wire.jsonl
```

## Data Models

```go
// provider/provider.go

type TokenUsage struct {
    InputOther       int `json:"input_other"`
    Output           int `json:"output"`
    InputCacheRead   int `json:"input_cache_read"`
    InputCacheCreate int `json:"input_cache_creation"`
}

func (t TokenUsage) TotalInput() int
func (t TokenUsage) Total() int

type SessionInfo struct {
    SessionID   string
    Title       string
    WorkDirHash string
    StartTime   time.Time
    EndTime     time.Time
    Turns       int
    TokenUsage  TokenUsage  // aggregated across all usage.record events
}

type DailyStats struct {
    Date       string        // "2026-02-17"
    Sessions   int
    TokenUsage TokenUsage
}

type Provider interface {
    Name() string
    CollectSessions(baseDir string) ([]SessionInfo, error)
}
```

## Acceptance Criteria

### Unit Tests

1. **kimi/parser_test.go**
   - TestParseWireJSONL_ValidData: parse fixture wire.jsonl, verify correct token counts
   - TestParseWireJSONL_EmptyFile: handle empty wire.jsonl gracefully
   - TestParseWireJSONL_MalformedLine: skip malformed JSON lines without crashing
   - TestParseWireJSONL_NoUsageRecord: wire.jsonl with no usage.record events returns zero tokens
   - TestParseSessionState_ValidData: parse state.json, verify title
   - TestParseSessionState_MissingFile: handle missing state.json gracefully
   - TestCollectSessions_MultipleSessionDirs: scan nested directory structure correctly
   - TestParseSession_MultiAgentAggregation: sum usage across agents, count only main agent turns
   - TestTimestampExtraction: verify start/end time from event timestamps

2. **stats/aggregator_test.go**
   - TestAggregateByDay_SingleDay: all sessions on same day
   - TestAggregateByDay_MultipleDays: sessions across different days
   - TestAggregateByDay_EmptySessions: no sessions returns empty result
   - TestAggregateByDay_DateFilter: filter by date range (--since/--until)

### E2E Tests

1. **e2e/e2e_test.go**
   - TestDailyCommand_JSONOutput: run `codetok daily --json --base-dir <testdata>`, verify JSON output structure
   - TestSessionCommand_JSONOutput: run `codetok session --json --base-dir <testdata>`, verify session list
   - TestDailyCommand_TableOutput: verify human-readable table output format
   - TestSessionCommand_TableOutput: verify human-readable session table
