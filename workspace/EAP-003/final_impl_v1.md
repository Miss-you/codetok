# EAP-003 Final Implementation v1

## Decision

Implement an opt-in range-aware usage event collection path that narrows provider candidate files before parsing while keeping the existing full-history API and final exact event timestamp filtering.

## API

- Add `provider.UsageEventCollectOptions` with `Since`, `Until`, and `Location`.
- Add optional `provider.UsageEventCollectMetrics` to count considered, skipped, parsed, and emitted events for tests and benchmarks.
- Add `provider.RangeAwareUsageEventProvider`.
- Add small option helpers for "has a range", localized inclusive timestamp inclusion, and file `ModTime` candidate checks.
- `Since` is the selected local start-of-day instant. `Until` is the selected local end-of-day instant. Providers that filter emitted rows must use the same localized inclusive date semantics as `stats.FilterEventsByDateRange`.
- File `ModTime` filtering is lower-bound only: skip a file only when `Since` is set and `ModTime` is safely before `Since`. Never skip JSONL, wire, or CSV files merely because `ModTime` is after `Until`.

## Command Changes

- Add `collectUsageEventsFromProvidersInRange`.
- Keep `collectUsageEventsFromProviders` as a no-options wrapper for existing tests and call sites.
- In `daily`, resolve the date window before collection, then call range-aware collection only when the resolved range is non-empty.
- In `session`, call range-aware collection only when `--since` or `--until` is set.
- Preserve the existing `stats.FilterEventsByDateRange` calls.

## Provider Changes

- Claude implements `CollectUsageEventsInRange` by collecting paths, filtering candidates by JSONL file `ModTime`, then parsing candidates.
- Codex implements `CollectUsageEventsInRange` by collecting paths, keeping files whose dated path could overlap the localized range with a one-day lower-bound lookback, and keeping any older file with `ModTime >= Since`.
- Kimi implements `CollectUsageEventsInRange` by collecting session paths, filtering candidates by `wire.jsonl` `ModTime`, then parsing candidates.
- Cursor implements `CollectUsageEventsInRange` by preserving CSV discovery and filtering emitted row events with the provided timestamp range. It must not skip whole CSV files by `ModTime`.

## Tests

- Add cmd collector tests proving range-aware providers receive options and legacy providers preserve fallback behavior.
- Add daily/session tests proving command date flags choose the expected collection path.
- Add provider tests for inactive-file skipping and cross-day in-window inclusion.
- Keep e2e cross-day tests as final smoke for user-facing behavior.
