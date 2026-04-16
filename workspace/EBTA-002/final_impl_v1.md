# EBTA-002 Final Implementation V1

## Selected Approach

Implement the narrow command bridge and timezone helper layer needed by later EBTA command tasks.

## Files

- `cmd/collect.go`: add usage-event collection helpers beside the existing session helpers.
- `cmd/collect_test.go`: add focused native-event and fallback-session tests.
- `cmd/daily.go`: add timezone resolution and local-date range helpers.
- `cmd/daily_test.go`: add focused timezone and local-window tests, updating existing date-range expectations where the helper signature changes.

## Collector Semantics

- Reuse the existing provider filter and directory override behavior.
- Prefer native `CollectUsageEvents` when available.
- Preserve native events exactly; do not rewrite timestamps, tokens, source path, event ID, model, or session metadata.
- For legacy providers, synthesize one event per session with `Timestamp=StartTime` and copied token/session metadata.
- Missing directories remain non-fatal.
- Errors stay provider-scoped and wrapped.

## Timezone Semantics

- Empty timezone means `time.Local`.
- Non-empty timezone must resolve through `time.LoadLocation`.
- Invalid timezone returns a concise `invalid --timezone` error.
- Explicit dates and default rolling windows are resolved in the selected location.

## Non-Goals

- Do not change `daily` aggregation from sessions to events.
- Do not change `session` command behavior.
- Do not add provider-native event parsers.
- Do not update README or e2e fixtures.

## OpenSpec

Use `event-based-token-aggregation-command-helpers` for EBTA-002. The change captures only the command bridge and timezone helper contract; it does not claim the later EBTA-007/008 switch to event aggregation.
