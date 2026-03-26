## Why

After Change 1 (`token-usage-output-split`) renames `TokenUsage.Output` to `TokenUsage.OutputOther` in `provider/provider.go`, all provider packages that construct `TokenUsage` values with the `Output:` field will fail to compile. The Claude, Kimi, and Cursor providers must be updated to use the new field name.

## What Changes

- Rename `Output:` to `OutputOther:` in every `provider.TokenUsage{}` literal across Claude, Kimi, and Cursor parsers
- Rename the internal `usageEntry.output` field to `usageEntry.outputOther` in Claude's parser (and its corresponding references)
- Update `usage.Output` accumulation to `usage.OutputOther` in Kimi's `parseWireJSONL`
- Update all test assertions from `.Output` to `.OutputOther` in the three provider test files
- No logic changes, no new fields populated; `OutputReasoning` stays at Go zero value (0)

## Capabilities

### New Capabilities

- `provider-output-rename`: Mechanical rename of `Output` to `OutputOther` across Claude, Kimi, and Cursor provider code and tests

### Modified Capabilities

(none -- no existing specs)

## Impact

- **Code**: `provider/claude/parser.go`, `provider/kimi/parser.go`, `provider/cursor/parser.go`
- **Tests**: `provider/claude/parser_test.go`, `provider/kimi/parser_test.go`, `provider/cursor/parser_test.go`
- **Dependencies**: Requires Change 1 (`token-usage-output-split`) to be applied first
- **Breaking**: None -- this is a follow-up rename that restores compilation after Change 1
