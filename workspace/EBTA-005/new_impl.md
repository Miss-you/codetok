# Candidate Implementation

Add native Kimi event collection without changing session parsing:

- Implement `CollectUsageEvents(baseDir string)` on `Provider`.
- Reuse the same session directory discovery and log fallback setup as `CollectSessions`.
- Add `parseUsageEvents(sessionPath, workDirHash, sessionModelIndex)` to parse metadata and scan `wire.jsonl`.
- Emit one `provider.UsageEvent` for each valid `StatusUpdate` with token usage.
- Use the `StatusUpdate` line timestamp, not `TurnBegin` or `TurnEnd`.
- Populate provider/session metadata using the same priority as `parseSession`.

The implementation stays provider-local. Command integration is explicitly deferred to EBTA-007 and EBTA-008.
