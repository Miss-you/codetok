# Codetok Kimi CLI Parser — Design Doc

## Data Source

Kimi CLI stores session data at `~/.kimi/sessions/<work-dir-hash>/<session-uuid>/`:
- `metadata.json`: session metadata (session_id, title, wire_mtime)
- `wire.jsonl`: event stream containing token usage in StatusUpdate events
- `context.jsonl`: conversation context (not needed for token stats)

### wire.jsonl Format

Each line is a JSON object. Event types:
- `metadata` (line 1): `{"type": "metadata", "protocol_version": "1.2"}`
- `TurnBegin`: `{"timestamp": ..., "message": {"type": "TurnBegin", "payload": {"user_input": [...]}}}`
- `StatusUpdate`: contains token_usage — **this is what we parse**
- `TurnEnd`: marks end of a turn
- Others: `StepBegin`, `ContentPart`, `ToolCall`, `ToolResult`

### StatusUpdate Payload (key data)

```json
{
  "timestamp": 1770983426.420942,
  "message": {
    "type": "StatusUpdate",
    "payload": {
      "context_usage": 0.024,
      "token_usage": {
        "input_other": 1562,
        "output": 66,
        "input_cache_read": 4864,
        "input_cache_creation": 0
      },
      "message_id": "chatcmpl-xxx"
    }
  }
}
```

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
│       ├── parser.go          # Kimi wire.jsonl parser
│       ├── parser_test.go     # Unit tests
│       └── testdata/          # Test fixtures
│           ├── wire.jsonl
│           └── metadata.json
├── stats/
│   ├── aggregator.go          # Aggregate by day/session
│   └── aggregator_test.go     # Unit tests
└── e2e/
    ├── e2e_test.go            # End-to-end CLI tests
    └── testdata/              # Full session fixture tree
        └── sessions/
            └── abc123/
                └── uuid-1/
                    ├── wire.jsonl
                    └── metadata.json
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
    TokenUsage  TokenUsage  // aggregated across all StatusUpdate events
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
   - TestParseWireJSONL_NoStatusUpdate: wire.jsonl with no StatusUpdate events returns zero tokens
   - TestParseMetadata_ValidData: parse metadata.json, verify session_id and title
   - TestParseMetadata_MissingFile: handle missing metadata.json gracefully
   - TestCollectSessions_MultipleSessionDirs: scan nested directory structure correctly
   - TestTimestampExtraction: verify start/end time from TurnBegin/TurnEnd timestamps

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
