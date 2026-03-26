## 1. Struct field changes

- [ ] 1.1 Rename `Output int` field to `OutputOther int` and change JSON tag from `"output"` to `"output_other"` in `TokenUsage` struct in `provider/provider.go`
- [ ] 1.2 Add `OutputReasoning int` field with JSON tag `"output_reasoning"` to `TokenUsage` struct in `provider/provider.go`

## 2. Method changes

- [ ] 2.1 Add `TotalOutput() int` method on `TokenUsage` that returns `OutputOther + OutputReasoning`
- [ ] 2.2 Update `Total()` method to return `TotalInput() + TotalOutput()` instead of `TotalInput() + Output`

## 3. Verification

- [ ] 3.1 Verify `provider/provider.go` compiles in isolation: `go build ./provider/`
- [ ] 3.2 Confirm that references to `TokenUsage.Output` in other packages produce expected compilation errors (this validates the breaking change is correctly scoped)
