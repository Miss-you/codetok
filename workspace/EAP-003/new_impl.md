# EAP-003 Proposed Implementation

## Shared Shape

Add a provider-level option and opt-in interface:

```go
type UsageEventCollectOptions struct {
    Since time.Time
    Until time.Time
    Location *time.Location
}

type RangeAwareUsageEventProvider interface {
    Provider
    CollectUsageEventsInRange(baseDir string, opts UsageEventCollectOptions) ([]UsageEvent, error)
}
```

`CollectUsageEvents` remains the full-history compatibility path. Command collection selects the range-aware method only when a range is present and the provider implements it.

## Command Adoption

- `daily` resolves timezone and date range before collection.
- `daily --all` passes no range and therefore keeps full collection.
- `session --since/--until` passes a range; `session` without date filters keeps full collection.
- Commands still run `stats.FilterEventsByDateRange` after provider collection for exact attribution.

## Provider Candidate Filtering

- Claude: skip JSONL files whose `ModTime` is before `Since`; include files modified in-window so cross-day sessions survive.
- Codex: use dated path layout and `ModTime`; include files whose path date is on or after `Since - 1 day`, or whose `ModTime` is on/after `Since`.
- Kimi: skip session directories only when `wire.jsonl` `ModTime` is before `Since`.
- Cursor: keep directory rules unchanged; range-aware collection may filter emitted row events by timestamp but must preserve explicit `--cursor-dir` semantics.

## Non-Goals

- No durable parsed-event cache.
- No direct-to-aggregator streaming in this task.
- No Codex JSON parser rewrite in this task.
