# EBTA-001 Test Strategy

## Focused Tests

Run:

```bash
go test ./stats -run TestAggregateEvents
```

This must prove:

- One session with events on two local dates yields two daily rows.
- The same timestamp maps to different dates under UTC and Asia/Shanghai style timezones.
- `DailyStats.Sessions` counts distinct sessions, not event count.
- Missing `SessionID` falls back to `SourcePath` without using `EventID`.
- Model grouping preserves existing provider metadata behavior.
- Event filtering includes only events whose localized date key falls inside `[since, until]`.

## Package Tests

Run:

```bash
go test ./provider ./stats
```

This confirms the new provider API compiles and existing stats/session aggregation tests still pass.

## Final Gates

Run the workflow-required gates:

```bash
make fmt
make test
make vet
make build
```

Run `make lint` only if `golangci-lint` is installed.
