# EBTA-007 Test Strategy

## Scope

Daily command integration only. Provider parsing, session command behavior, e2e acceptance, and README updates remain separate tasks.

## RED

Add command tests before production code:

```bash
go test -count=1 ./cmd -run 'Test(DailyJSON|DailyDateWindowValidationPrecedesCollection|BuildDailyStatsFromUsageEvents)'
```

Expected initial failures:

- command JSON test gets the wrong date or no row because `runDaily` still consumes sessions
- date-window validation test gets the provider collection error before the expected flag conflict error
- helper tests fail because `buildDailyStatsFromUsageEvents` does not exist

Command-level `runDaily` tests must feed deterministic data through an isolated fake `provider.UsageEventProvider` registered with a unique provider name and selected with `--provider`. The fake provider's `CollectSessions` should return a session-start-only shape that would fail the event-date assertions if `runDaily` still uses sessions.

Default `--days` local-midnight behavior should be tested through `resolveDailyDateRange` plus `buildDailyStatsFromUsageEvents` using a fixed `now`; do not rely on wall-clock `time.Now()` in `runDaily`.

## GREEN

Implement the minimal event pipeline in `cmd/daily.go`, then rerun:

```bash
go test -count=1 ./cmd -run 'Test(DailyJSON|DailyDateWindowValidationPrecedesCollection|BuildDailyStatsFromUsageEvents)'
go test -count=1 ./cmd -run 'Test(Daily|BuildDailyStatsFromUsageEvents)'
```

## Regression

After focused tests pass:

```bash
go test -count=1 ./cmd ./stats -run 'Test(Daily|AggregateEvents|FilterEvents)'
make fmt
make test
make vet
make build
make lint
```

`make lint` is required when `golangci-lint` is installed.
