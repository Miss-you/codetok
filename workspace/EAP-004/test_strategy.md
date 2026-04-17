# EAP-004 Test Strategy

## RED First

The task started with compile-failing tests for missing APIs:

- `go test ./cmd ./stats`
- Failure: `undefined: NewDailyEventAggregator`
- Failure: `undefined: aggregateDailyUsageEventsFromProvidersInRange`

## Focused Coverage

- `stats`: incremental aggregation matches `AggregateEventsByDayWithDimension`.
- `cmd`: direct daily aggregation matches materialized `FilterEventsByDateRange` plus `AggregateEventsByDayWithDimension`.
- `cmd`: provider errors keep existing context and wrap the root cause.
- `cmd`: `runDailyWithProviders --json` matches materialized stats.
- `cmd`: dashboard output matches materialized stats rendered through the existing dashboard printer.

## Verification Gates

Required:

- `go test ./cmd ./stats`
- `go test ./provider/...`
- `make fmt`
- `make test`
- `make vet`
- `make lint` when available
- `make build`

Manual built-binary smoke after build:

- `./bin/codetok daily --json`
- `./bin/codetok daily --all --json`
- `./bin/codetok daily --unit raw`

Static evidence:

- `cmd/daily.go` should not call `collectUsageEventsFromProvidersInRange`.
- `cmd/daily.go` should not build/filter an `allEvents` slice.
