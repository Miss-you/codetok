# EBTA-008 Test Strategy

## RED Evidence

After adding focused session tests:

```text
go test ./cmd -run 'Test(RunSession|AggregateSessionEvents)'
```

failed because `runSessionWithProviders` and `aggregateSessionEvents` did not exist.
This proves the new tests exercise behavior that was not implemented.

## Focused Coverage

- `TestRunSession_JSONFiltersByUsageEventDate` proves a cross-day session is included
  by in-range event date and totals only filtered events.
- `TestRunSession_JSONTimezoneFiltersByLocalEventDate` proves the `--timezone` date key
  controls session filtering and output date.
- `TestRunSession_InvalidTimezone` proves invalid timezone input is rejected.
- `TestAggregateSessionEventsTracksFirstAndLastIncludedEvents` proves grouping uses
  provider/session boundaries, sums events, preserves internal metadata, and tracks
  first/last included event timestamps.

## Gates

Run the narrow command tests first, then broader command/provider/stats tests, then
repository gates: `make fmt`, `make test`, `make vet`, `make build`, and `make lint`
when available.
