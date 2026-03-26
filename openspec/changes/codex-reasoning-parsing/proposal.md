## Why

The Codex provider's JSONL session data includes a `reasoning_output_tokens` field in `total_token_usage`, but the parser ignores it. All reasoning tokens are currently lumped into the general output count, making it impossible to distinguish reasoning from non-reasoning output. With the `TokenUsage` struct gaining `OutputOther` and `OutputReasoning` fields (from the token-usage-output-split change), the Codex parser needs to extract and map the reasoning token count.

## What Changes

- Add `ReasoningOutputTokens int` field to the `tokenCountInfo.TotalTokenUsage` struct to unmarshal `reasoning_output_tokens` from JSONL
- Update `token_count` event parsing to populate `OutputReasoning` from `tu.ReasoningOutputTokens` and compute `OutputOther` as `tu.OutputTokens - tu.ReasoningOutputTokens`
- Update unit tests to assert correct `OutputReasoning` and `OutputOther` values (existing test fixtures already contain `reasoning_output_tokens` values of 20, 50, and 100)

## Capabilities

### New Capabilities
- `codex-reasoning-parsing`: Parse reasoning output tokens from Codex CLI session JSONL data and map them to the `OutputReasoning` field of `TokenUsage`

### Modified Capabilities

(none -- no existing specs)

## Impact

- **Code**: `provider/codex/parser.go` -- `tokenCountInfo` struct and `token_count` parsing block
- **Tests**: `provider/codex/parser_test.go` -- update assertions for `OutputReasoning` and `OutputOther` (test fixtures already have the data)
- **Dependencies**: Assumes `provider.TokenUsage` already has `OutputOther` and `OutputReasoning` fields (from token-usage-output-split change)
- **No CLI or command changes**: output columns are handled separately by the report-reasoning-column change
