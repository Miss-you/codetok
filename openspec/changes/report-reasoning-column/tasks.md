## 1. Update mergeTokenUsage aggregation

- [ ] 1.1 Rename `dst.Output += src.Output` to `dst.OutputOther += src.OutputOther` in `mergeTokenUsage()` in `cmd/daily.go`
- [ ] 1.2 Add `dst.OutputReasoning += src.OutputReasoning` to `mergeTokenUsage()` in `cmd/daily.go`

## 2. Update daily share table

- [ ] 2.1 Add "Reasoning" column header to `printTopGroupShare()` header row, between "Output" and "Cache Read"
- [ ] 2.2 Add `OutputReasoning` value to each data row in `printTopGroupShare()`, between Output and Cache Read
- [ ] 2.3 Rename `.Output` to `.OutputOther` in the "Output" column data reference in `printTopGroupShare()`

## 3. Update session table

- [ ] 3.1 Add "Reasoning" column header to `printSessionTable()` header row, between "Output" and "Total"
- [ ] 3.2 Add `OutputReasoning` value to each session row in `printSessionTable()`, between Output and Total
- [ ] 3.3 Add `OutputReasoning` summation to the TOTAL row accumulation loop in `printSessionTable()`
- [ ] 3.4 Add `OutputReasoning` value to the TOTAL summary row output in `printSessionTable()`
- [ ] 3.5 Rename `.Output` to `.OutputOther` in session row and TOTAL row references in `printSessionTable()`

## 4. Update tests

- [ ] 4.1 Update `cmd/daily_test.go` test fixtures: rename `Output:` to `OutputOther:` in all `provider.TokenUsage{}` struct literals, and verify "Reasoning" column appears in share table output
- [ ] 4.2 Verify `make test` passes with all changes applied (requires Changes 1-3 to be applied first)
