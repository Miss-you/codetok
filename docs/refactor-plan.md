# Extensibility Refactoring + Claude Code & Codex CLI Support

## Current Problems

1. **cmd/ hardcodes Kimi provider**: `daily.go` and `session.go` directly import and instantiate `kimi.Provider{}`
2. **`--base-dir` flag is Kimi-specific**: should be per-provider or auto-resolved
3. **SessionInfo.ProviderName missing**: output doesn't show which tool a session came from
4. **No provider registry**: no way to discover/enable multiple providers

## Refactoring Priority Order

### Phase 1: Provider Registry + CLI Refactoring (must do first)

1. **Add `ProviderName` to `SessionInfo`** — so output can show which tool each session/day came from
2. **Create `provider/registry.go`** — a simple registry that returns all known providers
3. **Refactor `cmd/daily.go` and `cmd/session.go`** — iterate all registered providers, merge sessions, replace `--base-dir` with provider-specific `--kimi-dir`, `--claude-dir`, `--codex-dir` and a `--provider` flag to filter
4. **Add `ProviderName` to JSON output** — DailyStats and sessionJSON should include provider name
5. **Update table output** — add Provider column

### Phase 2: Claude Code Provider

Data location: `~/.claude/projects/<project-slug>/<session-uuid>.jsonl`

JSONL format (one JSON object per line):
- **type="user"**: user messages, contains `uuid`, `timestamp`, `sessionId`
- **type="assistant"**: assistant response, contains `message.usage`:
  ```json
  {
    "input_tokens": 3,
    "cache_creation_input_tokens": 9642,
    "cache_read_input_tokens": 21913,
    "output_tokens": 9
  }
  ```
- **type="system"**: system events
- Session ID from `sessionId` field on any message
- Timestamp from `timestamp` field (ISO 8601 string)

Mapping to our TokenUsage:
- `input_tokens` → InputOther
- `cache_read_input_tokens` → InputCacheRead
- `cache_creation_input_tokens` → InputCacheCreate
- `output_tokens` → Output

### Phase 3: Codex CLI Provider

Data location: `~/.codex/sessions/<year>/<month>/<day>/rollout-<timestamp>-<uuid>.jsonl`

JSONL format:
- **type="session_meta"**: session metadata with `payload.id`, `payload.timestamp`, `payload.cwd`, `payload.cli_version`
- **type="event_msg"**: events, subtypes include:
  - `user_message`: user input
  - `token_count`: **this is what we parse** — contains `payload.info.total_token_usage`:
    ```json
    {
      "input_tokens": 9311,
      "cached_input_tokens": 7680,
      "output_tokens": 143,
      "reasoning_output_tokens": 64,
      "total_tokens": 9454
    }
    ```
- **type="response_item"**: messages (user/developer/assistant)
- **type="turn_context"**: turn metadata with model name

For Codex, we take the LAST `token_count` event per session (it has `total_token_usage` which is cumulative).

Mapping to our TokenUsage:
- `input_tokens - cached_input_tokens` → InputOther
- `cached_input_tokens` → InputCacheRead
- `output_tokens` → Output
- `reasoning_output_tokens` → (new field or fold into Output)
- InputCacheCreate → 0 (Codex doesn't report this)

## Test Matrix

### Unit Tests — Claude Code Parser (provider/claude/parser_test.go)
- TestParseClaudeSession_ValidData
- TestParseClaudeSession_EmptyFile
- TestParseClaudeSession_MalformedLine
- TestParseClaudeSession_NoAssistantMessages
- TestCollectClaudeSessions_MultipleProjects

### Unit Tests — Codex Parser (provider/codex/parser_test.go)
- TestParseCodexSession_ValidData
- TestParseCodexSession_EmptyFile
- TestParseCodexSession_MalformedLine
- TestParseCodexSession_NoTokenCount
- TestParseCodexSession_MultipleTokenCounts (should take last cumulative)
- TestCollectCodexSessions_DateDirStructure

### Unit Tests — Registry (provider/registry_test.go)
- TestRegistryAllProviders
- TestRegistryFilterByName

### E2E Tests (e2e/)
- TestDailyCommand_MultiProvider_JSONOutput
- TestSessionCommand_MultiProvider_JSONOutput
- TestDailyCommand_ProviderFilter
- TestSessionCommand_ProviderFilter
