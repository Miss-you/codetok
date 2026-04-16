## Why

Kimi session parsing currently collapses all `StatusUpdate` token usage into one session total, which loses the timestamp needed for event-date daily aggregation. EBTA-005 needs Kimi to emit native `UsageEvent` records before daily/session commands can switch away from session-start attribution.

## What Changes

- Add native Kimi usage event collection through `Provider.CollectUsageEvents`.
- Emit one timestamped `UsageEvent` per valid `StatusUpdate` payload containing `token_usage`.
- Preserve Kimi metadata behavior for session ID, title, workdir hash, model name, and log fallback model lookup.
- Keep existing `CollectSessions` behavior compatible while adding event collection.

## Capabilities

### New Capabilities

- `kimi-native-usage-events`: Kimi provider emits timestamped local token usage events from `StatusUpdate` records.

### Modified Capabilities

- None.

## Impact

- Affected code: `provider/kimi/parser.go`, `provider/kimi/parser_test.go`.
- No new dependencies.
- No remote API calls.
- Future command tasks can consume Kimi events through the shared `UsageEventProvider` interface.
