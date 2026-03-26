## 1. Claude provider rename

- [ ] 1.1 In `provider/claude/parser.go`, rename `usageEntry.output` field to `usageEntry.outputOther`
- [ ] 1.2 In `provider/claude/parser.go`, update the `usageEntry` literal (around line 228-233) to use `outputOther:` instead of `output:`
- [ ] 1.3 In `provider/claude/parser.go`, update `usage.Output += u.output` to `usage.OutputOther += u.outputOther` (around line 253)
- [ ] 1.4 In `provider/claude/parser_test.go`, rename all `.Output` assertions to `.OutputOther`

## 2. Kimi provider rename

- [ ] 2.1 In `provider/kimi/parser.go`, update `usage.Output +=` to `usage.OutputOther +=` in `parseWireJSONL` (around line 210)
- [ ] 2.2 In `provider/kimi/parser_test.go`, rename all `.Output` assertions to `.OutputOther`

## 3. Cursor provider rename

- [ ] 3.1 In `provider/cursor/parser.go`, update `Output:` to `OutputOther:` in the `TokenUsage{}` literal in `parseUsageRecord` (around line 257)
- [ ] 3.2 In `provider/cursor/parser_test.go`, rename all `.Output` assertions to `.OutputOther`

## 4. Verification

- [ ] 4.1 Run `make build` to confirm compilation succeeds (requires Change 1 applied first)
- [ ] 4.2 Run `make test` to confirm all tests pass with `-race -cover`
