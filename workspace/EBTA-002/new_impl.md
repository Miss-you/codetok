# EBTA-002 Candidate Implementation

## Collector Bridge

Add command helpers in `cmd/collect.go`:

- `collectUsageEvents(cmd)` calls `collectUsageEventsFromProviders(cmd, provider.Registry())`.
- `collectUsageEventsFromProviders` mirrors `collectSessionsFromProviders` for provider filtering and directory selection.
- When a provider implements `provider.UsageEventProvider`, call `CollectUsageEvents(dir)` and append events unchanged.
- When a provider only implements `provider.Provider`, call `CollectSessions(dir)` and synthesize one `provider.UsageEvent` per session.

Fallback event mapping:

- `ProviderName`, `ModelName`, `SessionID`, `Title`, `WorkDirHash`, and `TokenUsage` copy from the session.
- `Timestamp` uses `SessionInfo.StartTime`.
- `SourcePath` and `EventID` remain empty because `SessionInfo` does not carry stable event or file identity.

Error handling should match session collection:

- Skip `os.IsNotExist` from either native event collection or fallback session collection.
- Wrap operational errors with provider context.

## Timezone Helpers

Add reusable daily helper behavior in `cmd/daily.go`:

- Register `--timezone` as an IANA timezone name, defaulting to the user's local timezone when empty.
- Add `resolveTimezone(name string) (*time.Location, error)`.
- Refactor `resolveDailyDateRange` to accept a `*time.Location`.
- Parse `--since` and `--until` with `time.ParseInLocation`.
- Anchor the default `--days` window to `now.In(loc)` at local midnight.

This prepares later command integration without moving `daily` to event aggregation in EBTA-002.

## OpenSpec

Use `event-based-token-aggregation-command-helpers` for EBTA-002. The change is intentionally narrow: it covers command-side event collection and daily timezone/date-window helpers, while later EBTA tasks own provider-native events and command event aggregation.
