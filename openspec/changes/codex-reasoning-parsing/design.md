## Context

The Codex CLI writes session data as JSONL files containing `token_count` events with a `total_token_usage` object. This object already includes `reasoning_output_tokens` alongside `output_tokens`. The current `tokenCountInfo` struct does not have a field for `reasoning_output_tokens`, so the JSON unmarshaller silently drops it. The parser maps `tu.OutputTokens` directly to the `Output` field (which will become `OutputOther` after the token-usage-output-split change).

Existing test fixtures at `provider/codex/testdata/` and inline test data already contain `reasoning_output_tokens` values (20, 50, 100), so no fixture changes are needed.

## Goals / Non-Goals

**Goals:**
- Parse `reasoning_output_tokens` from Codex `token_count` events
- Map reasoning tokens to `OutputReasoning` and compute `OutputOther` as the difference
- Maintain the cumulative last-entry-wins semantics for token counts

**Non-Goals:**
- Changing the `Provider` interface or `TokenUsage` struct (done by token-usage-output-split)
- Modifying CLI output columns or report formatting (done by report-reasoning-column)
- Touching any provider other than Codex

## Decisions

### Decision 1: Add field to existing struct

**Choice**: Add `ReasoningOutputTokens int \`json:"reasoning_output_tokens"\`` to the inner anonymous struct of `tokenCountInfo`. This is the minimal change since Go's JSON unmarshaller already handles unknown fields gracefully.

**Alternative considered**: Create a new struct or use `json.RawMessage` for flexible parsing. Rejected because the field has a stable schema in Codex output and a typed field is simpler and safer.

### Decision 2: Compute OutputOther by subtraction

**Choice**: Set `OutputOther = tu.OutputTokens - tu.ReasoningOutputTokens` and `OutputReasoning = tu.ReasoningOutputTokens`. The Codex data uses `output_tokens` as the total output count (inclusive of reasoning), so subtraction gives the non-reasoning portion.

**Alternative considered**: Use `output_tokens` as `OutputOther` directly and add `reasoning_output_tokens` on top. Rejected because Codex's `output_tokens` already includes reasoning tokens (verified against real data), so adding would double-count.

### Decision 3: No defensive clamping

**Choice**: Do not clamp `OutputOther` to a minimum of 0. If `reasoning_output_tokens > output_tokens` (which would indicate a data bug), the negative value will surface as a visible anomaly rather than silently masking bad data.

**Rationale**: This matches the existing pattern for `InputOther = tu.InputTokens - tu.CachedInputTokens` which also does not clamp.

## Risks / Trade-offs

- **[Risk] reasoning_output_tokens missing from older Codex versions**: If the field is absent, Go unmarshals it as 0. `OutputOther = output_tokens - 0 = output_tokens`, which is the correct fallback. No risk.

- **[Risk] Field semantics change in future Codex versions**: If Codex changes `output_tokens` to exclude reasoning tokens, the subtraction would produce incorrect results. Mitigation: this is unlikely given API conventions, and would be caught by test fixture updates.
