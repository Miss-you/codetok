## ADDED Requirements

### Requirement: Reasoning column in daily share table

The `printTopGroupShare()` function SHALL display a "Reasoning" column showing `OutputReasoning` token counts, positioned between the "Output" and "Cache Read" columns.

#### Scenario: Daily share table with reasoning tokens

- **WHEN** `printTopGroupShare()` renders the share table
- **THEN** the header row SHALL contain columns in this order: Rank, \<group column\> (CLI or Model depending on `--group-by`), Share, Sessions, Total, Input, Output, Reasoning, Cache Read, Cache Create
- **AND** the "Reasoning" column SHALL display `TokenUsage.OutputReasoning` formatted with the active token unit

#### Scenario: Daily share table with zero reasoning tokens

- **WHEN** a group has `OutputReasoning == 0`
- **THEN** the Reasoning column SHALL display "0" (or "0.00k" etc. per the active unit)

### Requirement: Reasoning column in session table

The `printSessionTable()` function SHALL display a "Reasoning" column showing `OutputReasoning` token counts, positioned between the "Output" and "Total" columns.

#### Scenario: Session table with reasoning tokens

- **WHEN** `printSessionTable()` renders the session table
- **THEN** the header row SHALL contain columns in this order: Date, Provider, Session, Title, Input, Output, Reasoning, Total
- **AND** the "Reasoning" column SHALL display `TokenUsage.OutputReasoning` for each session

#### Scenario: Session table TOTAL row includes reasoning

- **WHEN** `printSessionTable()` renders the TOTAL summary row
- **THEN** the Reasoning column SHALL display the sum of `OutputReasoning` across all sessions

### Requirement: mergeTokenUsage includes OutputReasoning

The `mergeTokenUsage()` function SHALL aggregate `OutputReasoning` from the source into the destination, ensuring that daily and group totals include reasoning tokens.

#### Scenario: Merging token usage with reasoning

- **WHEN** `mergeTokenUsage(dst, src)` is called with `src.OutputReasoning > 0`
- **THEN** `dst.OutputReasoning` SHALL be incremented by `src.OutputReasoning`

### Requirement: Output column uses OutputOther field

The "Output" column in both `printTopGroupShare()` and `printSessionTable()` SHALL reference `TokenUsage.OutputOther` (not the removed `.Output` field), displaying only non-reasoning output tokens.

#### Scenario: Output column shows non-reasoning output

- **WHEN** a report table renders the "Output" column
- **THEN** the value SHALL be sourced from `TokenUsage.OutputOther`
