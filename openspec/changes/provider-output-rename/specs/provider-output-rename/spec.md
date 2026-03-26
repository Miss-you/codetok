## ADDED Requirements

### Requirement: Claude provider uses OutputOther field
The Claude provider SHALL assign parsed output token counts to `TokenUsage.OutputOther` instead of the removed `TokenUsage.Output` field. The internal `usageEntry` struct SHALL use `outputOther` as its field name. `OutputReasoning` SHALL remain at zero.

#### Scenario: Claude session with assistant messages
- **WHEN** a Claude session JSONL file contains assistant messages with `output_tokens` usage data
- **THEN** the parsed `SessionInfo.TokenUsage.OutputOther` SHALL equal the sum of deduplicated `output_tokens` values
- **AND** `SessionInfo.TokenUsage.OutputReasoning` SHALL equal 0

#### Scenario: Claude deduplication preserves correct OutputOther totals
- **WHEN** a Claude session contains streaming duplicates of the same assistant message (same `messageId:requestId`)
- **THEN** only the final entry's `output_tokens` SHALL be counted in `OutputOther`

### Requirement: Kimi provider uses OutputOther field
The Kimi provider SHALL assign parsed output token counts to `TokenUsage.OutputOther` instead of the removed `TokenUsage.Output` field. `OutputReasoning` SHALL remain at zero.

#### Scenario: Kimi session with StatusUpdate events
- **WHEN** a Kimi `wire.jsonl` file contains `StatusUpdate` events with `token_usage.output` data
- **THEN** the parsed `TokenUsage.OutputOther` SHALL equal the sum of all `output` values from `StatusUpdate` payloads
- **AND** `TokenUsage.OutputReasoning` SHALL equal 0

### Requirement: Cursor provider uses OutputOther field
The Cursor provider SHALL assign parsed output token counts to `TokenUsage.OutputOther` instead of the removed `TokenUsage.Output` field. `OutputReasoning` SHALL remain at zero.

#### Scenario: Cursor CSV row with Output Tokens column
- **WHEN** a Cursor usage CSV row contains a value in the `Output Tokens` column
- **THEN** the parsed `SessionInfo.TokenUsage.OutputOther` SHALL equal that integer value
- **AND** `SessionInfo.TokenUsage.OutputReasoning` SHALL equal 0

### Requirement: Test assertions use OutputOther
All test files for Claude, Kimi, and Cursor providers SHALL assert on `TokenUsage.OutputOther` instead of `TokenUsage.Output`.

#### Scenario: Existing tests pass after rename
- **WHEN** `make test` is run after applying this change (with Change 1 already applied)
- **THEN** all tests in `provider/claude/`, `provider/kimi/`, and `provider/cursor/` SHALL pass
