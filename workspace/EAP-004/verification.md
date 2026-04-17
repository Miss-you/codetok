# EAP-004 Verification

## Baseline

- `go test ./...`
- Result: pass.
- Note: e2e dominated runtime at about 240s.

## RED

- `go test ./cmd ./stats`
- Result: fail as expected.
- Missing symbols:
  - `NewDailyEventAggregator`
  - `aggregateDailyUsageEventsFromProvidersInRange`

## Focused GREEN

- `go test -count=1 ./stats -run TestDailyEventAggregator`
- Result: pass.
- `go test -count=1 ./cmd -run 'TestAggregateDailyUsageEventsFromProvidersInRange|TestRunDaily_JSONUsesStreaming|TestRunDaily_DashboardUsesStreaming'`
- Result: pass.
- `go test -count=1 ./provider/...`
- Result: pass.
- `go test -count=1 ./cmd -run 'TestCollectUsageEventsFromProviders|TestRunDaily|TestRunSession'`
- Result: pass.
- `go test -count=1 ./e2e -run TestEventBasedCrossDayAcceptance`
- Result: pass.

## Static Evidence

Command:

```bash
rg -n 'collectUsageEventsFromProvidersInRange|allEvents|FilterEventsByDateRange\(allEvents|AggregateEventsByDayWithDimension\(allEvents' cmd/daily.go || true
```

Result: no matches.

Meaning: `daily` no longer builds or filters an all-provider `allEvents` slice.

## Allocation Evidence

Command:

```bash
go test -run '^$' -bench BenchmarkDailyAggregationMaterializedVsStreaming -benchmem ./cmd
```

Result:

```text
BenchmarkDailyAggregationMaterializedVsStreaming/materialized-10     9948878 ns/op  25470600 B/op  100081 allocs/op
BenchmarkDailyAggregationMaterializedVsStreaming/streaming-10        8022923 ns/op   2410068 B/op  100076 allocs/op
```

Meaning: the streaming daily path removes the large command-level event-slice allocations in the synthetic benchmark. Provider-level parser allocations are intentionally out of scope for EAP-004.

## Final Gates

- `make fmt`: pass.
- `make test`: pass.
- `make vet`: pass.
- `make lint`: pass (`0 issues`).
- `make build`: pass.
- `git diff --check`: pass.

## Built Binary Smoke

Commands:

```bash
./bin/codetok daily --json >/tmp/codetok-eap004-daily.json
./bin/codetok daily --all --json >/tmp/codetok-eap004-daily-all.json
./bin/codetok daily --unit raw >/tmp/codetok-eap004-daily-raw.txt
```

Result:

```text
    4122 /tmp/codetok-eap004-daily.json
   59791 /tmp/codetok-eap004-daily-all.json
     849 /tmp/codetok-eap004-daily-raw.txt
   64762 total
```
