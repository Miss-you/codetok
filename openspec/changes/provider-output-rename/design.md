## Context

Change 1 (`token-usage-output-split`) renames `TokenUsage.Output` to `TokenUsage.OutputOther` and adds `TokenUsage.OutputReasoning` in `provider/provider.go`. After that rename, three providers -- Claude, Kimi, and Cursor -- reference the old `Output` field name and will not compile. This change performs the mechanical rename in those providers to restore compilation.

All three providers produce only non-reasoning output tokens. None of them have access to reasoning token data, so `OutputReasoning` remains at Go's zero value (0) for all of them.

## Goals / Non-Goals

**Goals:**
- Rename all references to `TokenUsage.Output` to `TokenUsage.OutputOther` in Claude, Kimi, and Cursor provider packages
- Update internal helper structs (Claude's `usageEntry.output`) to match the new naming
- Update all test assertions to reference `.OutputOther` instead of `.Output`
- Maintain identical runtime behavior -- no logic changes

**Non-Goals:**
- Populating `OutputReasoning` in any of these providers (they lack reasoning token data)
- Changing JSON field tags in `statusPayload` or `claudeUsage` structs (these are source data mappings, not output field names)
- Touching `provider/provider.go` (done in Change 1)
- Touching `provider/codex/` (done in Change 2)
- Touching `cmd/` or `stats/` (done in Change 4)

## Decisions

### Decision 1: Rename internal struct fields to match

**Choice**: Rename Claude's `usageEntry.output` to `usageEntry.outputOther` for consistency with the `TokenUsage` field it maps to.

**Rationale**: Keeping internal naming aligned with the external struct reduces cognitive overhead. Since this is a private struct with only a few references, the rename is trivial.

### Decision 2: Leave JSON deserialization field names unchanged

**Choice**: The `claudeUsage.OutputTokens` (`json:"output_tokens"`), `statusPayload.TokenUsage.Output` (`json:"output"`), and CSV column header `"Output Tokens"` remain unchanged.

**Rationale**: These names reflect the upstream data format. Renaming them would break parsing. The rename only affects the Go struct field in `provider.TokenUsage` where the parsed value is stored.

### Decision 3: Do not set OutputReasoning

**Choice**: Leave `OutputReasoning` at its zero value in all three providers.

**Rationale**: Claude, Kimi, and Cursor providers do not currently have reasoning token data in their source formats. If a provider later gains reasoning token support, it will be added in a separate change.

## Risks / Trade-offs

- **[Risk] Incomplete rename**: A missed reference would cause a compile error. --> **Mitigation**: `make build` immediately catches any missed field references. The rename is fully mechanical and grep-verifiable.

- **[Risk] Ordering dependency on Change 1**: This change only compiles after Change 1 is applied. --> **Mitigation**: Document the dependency. If applied out of order, `make build` will fail with clear errors pointing to the `TokenUsage` struct.
