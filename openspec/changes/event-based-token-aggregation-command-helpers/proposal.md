## Why

Command code needs a migration bridge from session-level provider output to event-level provider output before `daily` and `session` can switch to event-based aggregation. Date-window helpers also need to resolve user-visible calendar days in the selected timezone instead of assuming UTC.

## What Changes

- Add command collection helpers that prefer native `provider.UsageEventProvider` output.
- Fall back to one synthetic `provider.UsageEvent` per legacy `provider.SessionInfo` while providers migrate.
- Add `daily --timezone` parsing with local timezone as the default and concise invalid-timezone errors.
- Resolve `daily` `--since`, `--until`, and default `--days` windows in the selected timezone.

## Capabilities

### New Capabilities

- `event-based-token-aggregation-command-helpers`: Command-layer helper behavior for collecting usage events and resolving daily date windows by timezone.

### Modified Capabilities

- None.

## Impact

- Affected code: `cmd/collect.go`, `cmd/collect_test.go`, `cmd/daily.go`, `cmd/daily_test.go`.
- No provider parser changes.
- No remote API calls.
- No switch from session-start daily aggregation to event-date aggregation in this task; later EBTA tasks own that behavior change.
