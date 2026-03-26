## Why

After splitting `TokenUsage.Output` into `OutputOther` and `OutputReasoning` (Change 1) and updating all providers to populate these fields (Changes 2-3), the CLI report tables still display a single "Output" column referencing the old `.Output` field. Users cannot see how many tokens were consumed by model reasoning versus regular output, making it impossible to understand reasoning-heavy cost drivers from the reports.

## What Changes

- Add a "Reasoning" column to the daily `printTopGroupShare()` table, positioned between "Output" and "Cache Read"
- Add a "Reasoning" column to the session `printSessionTable()` table, positioned between "Output" and "Total"
- Update `mergeTokenUsage()` to include `OutputReasoning` in aggregation
- Rename all `.Output` references to `.OutputOther` in report formatting code to match the renamed struct field from Change 1
- Update TOTAL row in session table to sum `OutputReasoning`

## Capabilities

### New Capabilities

- `report-reasoning-column`: Display reasoning token counts as a dedicated column in daily and session report tables, with proper aggregation and totaling

### Modified Capabilities

(none -- no existing specs in openspec/specs/)

## Impact

- **Code**: `cmd/daily.go` -- `mergeTokenUsage()`, `printTopGroupShare()` formatting
- **Code**: `cmd/session.go` -- `printSessionTable()` formatting and TOTAL row
- **Tests**: `cmd/daily_test.go` -- update any assertions that reference column headers or Output field
- **Dependencies**: Requires Change 1 (token-usage-output-split) to be applied first so `OutputOther` and `OutputReasoning` fields exist on `TokenUsage`
