# EBTA-007 Final Implementation v1

## Decision

Use the existing event bridge and event stats helpers in the `daily` command. Do not introduce new aggregation rules in `cmd`.

## Code plan

1. Add focused tests in `cmd/daily_test.go`:
   - same-session usage events crossing two UTC dates produce two daily JSON rows
   - `--timezone Asia/Shanghai` shifts an event's date key from UTC evening to the next local day
   - default `--days 1` uses the selected local midnight when converted to event date keys
   - model grouping preserves `provider`, `group_by`, `group`, and `providers` semantics after the event switch
   - invalid date-window flag combinations return flag errors before provider collection errors
2. Update `cmd/daily.go`:
   - resolve the daily date window before collecting provider data, so flag validation remains authoritative
   - `collectUsageEvents(cmd)` replaces `collectSessions(cmd)`
   - `buildDailyStatsFromUsageEvents` converts date bounds to keys, filters events, and delegates aggregation to `stats`
   - `dailyDateBound` handles zero bounds safely
3. Keep validation order compatible:
   - invalid `--timezone` and invalid daily date-window combinations fail before collection
   - JSON output continues to ignore invalid dashboard-only `--unit` and `--top`
   - non-JSON dashboard output still validates `--unit` and `--top`

## OpenSpec

Use `event-based-token-aggregation-daily` for this command behavior change. The change describes only daily command event aggregation.

## Verification

Focused red/green:

```bash
go test -count=1 ./cmd -run 'Test(DailyJSON|DailyDateWindowValidationPrecedesCollection|BuildDailyStatsFromUsageEvents)'
```

Task gate:

```bash
go test ./cmd -run 'Test(Daily|BuildDailyStatsFromUsageEvents)'
```

Broader gates:

```bash
make fmt
make test
make vet
make build
make lint
```

Run `make lint` when `golangci-lint` is available.
