## 1. Update tokenCountInfo struct

- [ ] 1.1 Add `ReasoningOutputTokens int \`json:"reasoning_output_tokens"\`` field to the `TotalTokenUsage` inner struct in `tokenCountInfo` in `provider/codex/parser.go`

## 2. Update token_count parsing logic

- [ ] 2.1 Change the `token_count` case in `parseCodexSession()` to set `OutputReasoning = tu.ReasoningOutputTokens` and `OutputOther = tu.OutputTokens - tu.ReasoningOutputTokens` (replacing the current `Output: tu.OutputTokens` assignment)

## 3. Update unit tests

- [ ] 3.1 Update `TestParseCodexSession_ValidData` assertions: replace `Output` check with `OutputOther` (250) and add `OutputReasoning` (50) assertion (fixture has `output_tokens=300, reasoning_output_tokens=50`)
- [ ] 3.2 Update `TestParseCodexSession_MalformedLine` assertions: replace `Output` check with `OutputOther` (80) and add `OutputReasoning` (20) assertion (inline data has `output_tokens=100, reasoning_output_tokens=20`)
- [ ] 3.3 Update `TestParseCodexSession_MultipleTokenCounts` assertions: replace `Output` check with `OutputOther` (700) and add `OutputReasoning` (100) assertion (last cumulative event has `output_tokens=800, reasoning_output_tokens=100`)
- [ ] 3.4 Add a new test `TestParseCodexSession_NoReasoningTokens` that uses inline JSONL data without `reasoning_output_tokens` field and asserts `OutputReasoning == 0` and `OutputOther == output_tokens`
- [ ] 3.5 Update `TestCollectCodexSessions_DateDirStructure` assertions: add `OutputOther` and `OutputReasoning` checks (inline data has `output_tokens=200, reasoning_output_tokens=50`, so `OutputOther=150, OutputReasoning=50`)

## 4. Verification

**Prerequisite**: Change `token-usage-output-split` must be applied first (provides `OutputOther` and `OutputReasoning` fields on `TokenUsage`).

- [ ] 4.1 Run `make test` to verify all tests pass with `-race -cover`
- [ ] 4.2 Run `make vet` to verify no static analysis issues
