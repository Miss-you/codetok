## Context

The `TokenUsage` struct in `provider/provider.go` is the central type used by all providers and all reporting commands. It currently has four fields: `InputOther`, `Output`, `InputCacheRead`, and `InputCacheCreate`. The `Output` field conflates regular output tokens with reasoning/thinking tokens. AI providers (Claude, Codex) now report reasoning tokens separately, so the struct needs to split output into two fields to support accurate per-category reporting.

This is Change 1 of 4 in a series. It intentionally breaks consumers to establish the new type shape. Subsequent changes fix the breakage in provider parsers, report commands, and tests.

## Goals / Non-Goals

**Goals:**
- Split `Output` into `OutputOther` and `OutputReasoning` in the `TokenUsage` struct
- Add a `TotalOutput()` method symmetric with the existing `TotalInput()` method
- Update `Total()` to compose `TotalInput() + TotalOutput()`
- Maintain the same arithmetic behavior: for providers that do not report reasoning tokens, `OutputReasoning` defaults to zero, so `TotalOutput()` equals the old `Output` value

**Non-Goals:**
- Updating any provider parsers to populate `OutputReasoning` (handled by Change 3)
- Updating any CLI report code to display reasoning columns (handled by Change 4)
- Renaming `Output` references in provider parsers (handled by Change 2)
- Adding input reasoning tokens (not currently reported by any supported provider)

## Decisions

### Decision 1: Rename Output to OutputOther (not keep both)

**Choice**: Rename the existing `Output` field to `OutputOther` to mirror the `InputOther` naming convention, and change the JSON tag to `"output_other"`.

**Alternative considered**: Keep `Output` as-is and only add `OutputReasoning`. Rejected because `Output` would then be ambiguous -- does it mean "all output" or "non-reasoning output"? The `InputOther`/`OutputOther` symmetry makes the semantics clear.

### Decision 2: Add TotalOutput() method

**Choice**: Add `TotalOutput() int` returning `OutputOther + OutputReasoning`, symmetric with the existing `TotalInput()` method.

**Rationale**: This gives callers a clean way to get total output without knowing the internal breakdown. It also simplifies `Total()` to `TotalInput() + TotalOutput()`.

### Decision 3: Accept intentional compilation breakage

**Choice**: Make this a breaking change in one step rather than a backward-compatible migration with deprecation.

**Rationale**: This is an internal type with no external consumers (codetok is a standalone CLI tool). All call sites are in the same repository and will be fixed by Changes 2-4 in the same PR or series. A deprecation cycle adds unnecessary complexity.

## Risks / Trade-offs

- **[Risk] Compilation breakage across the codebase**: After this change, all references to `TokenUsage.Output` and any code computing total output manually will fail to compile. --> **Mitigation**: This is intentional and scoped. Changes 2-4 fix all breakages. All changes should be merged together or in sequence.

- **[Trade-off] JSON schema change**: The `"output"` JSON key becomes `"output_other"`. Any persisted JSON output from previous versions will not match the new schema. --> **Acceptable**: codetok reads source provider JSONL files (not its own output), so there is no backward compatibility concern for input parsing. Output JSON format changes are expected as the tool evolves.
