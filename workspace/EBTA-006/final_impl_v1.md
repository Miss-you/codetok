# EBTA-006 Final Implementation v1

## Scope

Implement native Cursor usage events without changing Cursor CSV parsing, default source discovery, explicit directory behavior, or sync/cache code.

## Implementation

- Add `func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error)` in `provider/cursor/parser.go`.
- Reuse `resolveCursorCSVPaths(baseDir)` and `parseUsageCSV(path)`.
- Skip invalid CSV files exactly like `CollectSessions`.
- Convert each parsed row to one `provider.UsageEvent`:
  - `ProviderName`, `ModelName`, `SessionID`, `Title`, and `TokenUsage` come from the existing `SessionInfo`.
  - `Timestamp` uses `SessionInfo.StartTime`.
  - `SourcePath` is the CSV path.
  - `EventID` is the row-based `SessionID`, preserving existing basename/row identity instead of making event identity depend on caller path spelling.
- Sort events by `Timestamp`, then `SessionID`, matching the existing session collector ordering semantics.

## OpenSpec

No new OpenSpec delta for EBTA-006. The shared `UsageEvent` API is already defined by `event-based-token-aggregation-core`; this task is provider-only compatibility work and does not switch user-facing command behavior.

## Multi-Agent Review Notes

- Cursor semantics explorer confirmed current CSV/session behavior and recommended additive provider-only implementation.
- Test-strategy explorer confirmed no `cursor/` package edits should be required and proposed focused provider event tests plus `go test ./provider/cursor ./cursor`.
- Code-quality reviewer flagged path-spelling-dependent `EventID`; fixed by using the existing row identity.
- Owner review after fix: no high-severity issues. Estimated fit score 94/100 against CLI/provider semantics, Go simplicity, scope control, and testability.
