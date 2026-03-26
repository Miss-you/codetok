## Context

The `TokenUsage` struct (after Change 1) has `OutputOther` and `OutputReasoning` fields replacing the single `Output` field, plus a `TotalOutput()` method that sums both. The report layer in `cmd/daily.go` and `cmd/session.go` currently references `.Output` and does not display reasoning tokens. This change updates the report formatting to surface the new field.

The `dayTotal` and `groupTotal` structs embed `provider.TokenUsage`, so they automatically gain the new fields once Change 1 is applied. The only manual work is in `mergeTokenUsage()` (which explicitly copies individual fields) and the `fmt.Fprintf` formatting calls.

## Goals / Non-Goals

**Goals:**
- Show reasoning tokens in both daily and session report tables
- Maintain correct aggregation by including `OutputReasoning` in `mergeTokenUsage()`
- Keep the "Output" column showing only non-reasoning output (now `.OutputOther`)
- Preserve existing column ordering conventions (new column inserted logically between Output and Cache)

**Non-Goals:**
- Changing JSON output format (JSON already reflects `TokenUsage` struct via json tags)
- Adding reasoning-specific CLI flags or filtering
- Modifying the Daily Trend or Group Ranking sections (they use `Total()` which already includes reasoning via `TotalOutput()`)

## Decisions

### Decision 1: Column placement in daily share table

**Choice**: Insert "Reasoning" between "Output" and "Cache Read" in `printTopGroupShare()`.

**Rationale**: This groups all output-related columns together (Output, Reasoning) before input-related cache columns. The column order becomes: Rank, CLI, Share, Sessions, Total, Input, Output, Reasoning, Cache Read, Cache Create.

**Alternative considered**: Place Reasoning after Cache Create at the end. Rejected because it separates output-related metrics.

### Decision 2: Column placement in session table

**Choice**: Insert "Reasoning" between "Output" and "Total" in `printSessionTable()`.

**Rationale**: The session table has fewer columns and a simpler layout. Placing Reasoning next to Output keeps related metrics adjacent. The column order becomes: Date, Provider, Session, Title, Input, Output, Reasoning, Total.

### Decision 3: Use `.OutputOther` for the "Output" column

**Choice**: The "Output" column header remains "Output" but references `.OutputOther` instead of `.Output`. This aligns with Change 1 where `.Output` is renamed to `.OutputOther`.

**Rationale**: The column still represents non-reasoning output tokens. The header "Output" is user-friendly; the internal field name change is transparent to users.

### Decision 4: Update mergeTokenUsage()

**Choice**: Rename `dst.Output += src.Output` to `dst.OutputOther += src.OutputOther`, and add `dst.OutputReasoning += src.OutputReasoning` to the existing `mergeTokenUsage()` function.

**Rationale**: This function explicitly copies each field (it does not use reflection or struct assignment). The existing field must be renamed and the new field must be added to maintain correct aggregation.

## Risks / Trade-offs

- **[Risk] Column width in narrow terminals**: Adding a "Reasoning" column increases table width. -> **Mitigation**: The column uses the same `formatTokenByUnit()` formatting as other columns, keeping width consistent. The tabwriter handles alignment automatically.

- **[Risk] Compile error if Change 1 not applied**: References to `.OutputOther` and `.OutputReasoning` will fail to compile without the struct changes from Change 1. -> **Mitigation**: This is Change 4 of 4 and explicitly depends on Change 1 being applied first.

- **[Trade-off] No reasoning column in Daily Trend or Group Ranking sections**: These sections show only Total, which already includes reasoning via `TotalOutput()`. Adding per-column breakdown would clutter the trend view. -> **Acceptable**: The share table provides the detailed breakdown.
