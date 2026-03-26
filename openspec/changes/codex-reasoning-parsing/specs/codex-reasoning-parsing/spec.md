## ADDED Requirements

### Requirement: Parse reasoning output tokens from Codex JSONL
The Codex provider SHALL parse the `reasoning_output_tokens` field from `total_token_usage` objects in `token_count` events and map it to `TokenUsage.OutputReasoning`.

#### Scenario: Session with reasoning tokens
- **WHEN** a Codex `token_count` event contains `"reasoning_output_tokens": 50` and `"output_tokens": 300`
- **THEN** `OutputReasoning` SHALL be 50 and `OutputOther` SHALL be 250

#### Scenario: Session without reasoning tokens
- **WHEN** a Codex `token_count` event contains `"output_tokens": 100` and no `reasoning_output_tokens` field
- **THEN** `OutputReasoning` SHALL be 0 and `OutputOther` SHALL be 100

#### Scenario: Multiple cumulative token_count events
- **WHEN** a Codex session contains multiple `token_count` events with increasing cumulative values
- **THEN** only the last `token_count` event's `reasoning_output_tokens` and `output_tokens` SHALL be used (last-entry-wins)

### Requirement: Backward compatibility with existing data
The Codex provider SHALL continue to correctly parse session files that do not contain a `reasoning_output_tokens` field. Missing fields SHALL default to zero.

#### Scenario: Legacy session file without reasoning field
- **WHEN** a Codex session JSONL file contains `token_count` events with only `input_tokens`, `cached_input_tokens`, `output_tokens`, and `total_tokens`
- **THEN** `OutputReasoning` SHALL be 0 and `OutputOther` SHALL equal `output_tokens`
