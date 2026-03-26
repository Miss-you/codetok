## Why

The `TokenUsage` struct currently has a single `Output` field that combines all output tokens. AI model APIs now distinguish between regular output tokens and reasoning/thinking output tokens (e.g., Claude's extended thinking, Codex's reasoning tokens). Without splitting output into `OutputOther` and `OutputReasoning`, codetok cannot report reasoning token consumption separately, which is important because reasoning tokens often have different pricing and represent a distinct usage category.

## What Changes

- **BREAKING**: Rename `TokenUsage.Output` field to `OutputOther` and change its JSON tag from `"output"` to `"output_other"`
- Add new field `OutputReasoning int` with JSON tag `"output_reasoning"` to `TokenUsage`
- Add new method `TotalOutput() int` that returns `OutputOther + OutputReasoning`
- Update existing `Total()` method to use `TotalInput() + TotalOutput()` instead of `TotalInput() + Output`

## Capabilities

### New Capabilities

- `token-usage-output-split`: Split the single output token field into `OutputOther` and `OutputReasoning`, add `TotalOutput()` method, and update `Total()` to use it

### Modified Capabilities

(none -- no existing specs)

## Impact

- **Code**: `provider/provider.go` -- struct definition and methods only
- **Breaking consumers**: All code referencing `TokenUsage.Output` will fail to compile after this change. This is intentional; consumer updates are handled by separate changes (provider parsers, report commands)
- **JSON serialization**: The `"output"` JSON key is replaced by `"output_other"` and `"output_reasoning"` is added. Any stored or exported JSON using the old schema will need migration
- **Dependencies**: None -- uses only stdlib
