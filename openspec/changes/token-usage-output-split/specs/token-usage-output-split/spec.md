## ADDED Requirements

### Requirement: OutputOther field replaces Output

The `TokenUsage` struct SHALL have a field `OutputOther int` with JSON tag `"output_other"` in place of the former `Output int` field with JSON tag `"output"`. The field `Output` SHALL NOT exist.

#### Scenario: Struct field name and JSON tag
- **GIVEN** the `TokenUsage` struct in `provider/provider.go`
- **THEN** the struct SHALL contain `OutputOther int` with JSON tag `json:"output_other"`
- **AND** the struct SHALL NOT contain a field named `Output`

### Requirement: OutputReasoning field

The `TokenUsage` struct SHALL have a field `OutputReasoning int` with JSON tag `"output_reasoning"`. Its zero value (0) indicates no reasoning tokens were consumed.

#### Scenario: Struct field exists
- **GIVEN** the `TokenUsage` struct in `provider/provider.go`
- **THEN** the struct SHALL contain `OutputReasoning int` with JSON tag `json:"output_reasoning"`

### Requirement: TotalOutput method

The `TokenUsage` type SHALL have a method `TotalOutput() int` that returns the sum of `OutputOther` and `OutputReasoning`.

#### Scenario: Both output fields populated
- **GIVEN** a `TokenUsage` with `OutputOther = 100` and `OutputReasoning = 50`
- **WHEN** `TotalOutput()` is called
- **THEN** it SHALL return `150`

#### Scenario: Only OutputOther populated
- **GIVEN** a `TokenUsage` with `OutputOther = 200` and `OutputReasoning = 0`
- **WHEN** `TotalOutput()` is called
- **THEN** it SHALL return `200`

#### Scenario: Only OutputReasoning populated
- **GIVEN** a `TokenUsage` with `OutputOther = 0` and `OutputReasoning = 300`
- **WHEN** `TotalOutput()` is called
- **THEN** it SHALL return `300`

### Requirement: Total method uses TotalOutput

The `Total()` method SHALL return `TotalInput() + TotalOutput()`.

#### Scenario: Total includes both input and output categories
- **GIVEN** a `TokenUsage` with `InputOther = 10`, `InputCacheRead = 20`, `InputCacheCreate = 30`, `OutputOther = 40`, `OutputReasoning = 50`
- **WHEN** `Total()` is called
- **THEN** it SHALL return `150` (10 + 20 + 30 + 40 + 50)

#### Scenario: Total with no reasoning tokens equals legacy behavior
- **GIVEN** a `TokenUsage` with `InputOther = 100`, `InputCacheRead = 0`, `InputCacheCreate = 0`, `OutputOther = 200`, `OutputReasoning = 0`
- **WHEN** `Total()` is called
- **THEN** it SHALL return `300` (100 + 200)
