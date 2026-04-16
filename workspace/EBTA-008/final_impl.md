# EBTA-008 Final Implementation

## Decision

Implement the session event pipeline exactly in `cmd/session.go`, backed by focused
command tests in `cmd/session_test.go` and OpenSpec change
`event-based-token-aggregation-session-command`.

## Review Fixes Applied To Plan

- Keep session JSON schema stable; do not add a `model` field in EBTA-008.
- Cover table output so localized dates are not JSON-only.
- Cover fallback grouping for events without `SessionID`.
- Cover collector routing through provider filters and provider directory overrides.
- Cover invalid `--since` and `--until` validation.

## Implementation

`session` will:

1. resolve `--timezone`;
2. validate `--since` and `--until` as local date keys;
3. collect usage events through `collectUsageEventsFromProviders`;
4. filter events with `stats.FilterEventsByDateRange`;
5. aggregate filtered events by provider/session into deterministic `SessionInfo` rows;
6. render existing JSON/table output with dates in the selected timezone.

`Turns` remains a best-effort count of included usage events because `UsageEvent` has
no richer turn model.
