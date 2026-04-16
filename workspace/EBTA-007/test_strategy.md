# EBTA-007 Test Strategy

## Required Proof

- `daily` uses native usage events instead of session start times.
- Event timestamps are localized with `--timezone` before date grouping.
- Explicit `--since` and `--until` select events by localized date.
- Default `--days` starts at the selected timezone's local midnight.
- JSON grouping fields remain stable for CLI and model grouping.
- Existing daily flag constraints and dashboard behavior remain green.

## Focused Tests

- `TestRunDaily_JSONAggregatesUsageEventsByEventDate`
- `TestRunDaily_JSONTimezoneChangesEventDateKeys`
- `TestRunDaily_DefaultWindowFiltersByLocalEventDate`
- `TestRunDaily_ExplicitDateRangeFiltersByLocalEventDate`
- `TestRunDaily_JSONModelGroupingUsesUsageEventsAcrossProviders`

## Gates

- `go test ./cmd -run TestRunDaily`
- `go test ./cmd -run TestDaily`
- `go test ./provider ./stats ./cmd -run 'Test(CollectUsageEvents|AggregateEvents|FilterEvents|ResolveDaily|ResolveTimezone|RunDaily)'`
- `make fmt`
- `make test`
- `make vet`
- `make build`
- `make lint` when `golangci-lint` is installed
