# EAP-004 Original Implementation

## Current Flow

After EAP-003, `cmd/daily.go` resolves the selected date window before provider collection. `runDailyWithProviders` then called `collectUsageEventsFromProvidersInRange`, converted the resolved time bounds to date strings, filtered with `stats.FilterEventsByDateRange`, and aggregated with `stats.AggregateEventsByDayWithDimension`.

`cmd/collect.go` collected providers sequentially. For native usage-event providers it called `CollectUsageEvents` or `CollectUsageEventsInRange` and appended each provider slice into one command-level `allEvents` slice. For legacy providers it converted sessions into synthetic usage events and appended them to the same slice.

## Materialization Points

- `cmd/collect.go` built one cross-provider `[]provider.UsageEvent`.
- `stats.FilterEventsByDateRange` built a second filtered `[]provider.UsageEvent`.
- Provider APIs still return `[]provider.UsageEvent`, so providers may materialize per-provider or per-file events internally.

## Existing Contracts

- Command-level provider errors are wrapped with provider context, for example `collecting usage events from <provider>`.
- Missing local provider roots are skipped.
- Range-aware providers may emit candidate events outside the final daily window; exact event timestamp filtering stays authoritative.
- JSON and dashboard rendering consume `[]provider.DailyStats`.

## Streaming Constraints

EAP-004 should remove the command-level all-provider event slice and filtered event slice from `daily`, but should not rewrite provider parsers. Provider-level streaming would need parser-specific work because Claude deduplicates per file, Codex computes cumulative deltas, and Kimi applies metadata/model fallback.
