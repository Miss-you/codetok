# EBTA-002 Test Strategy

## Focused Collector Tests

Run:

```bash
go test ./cmd -run TestCollectUsageEvents
```

This must prove:

- A fake native provider implementing `provider.UsageEventProvider` is collected through `CollectUsageEvents`.
- Native events are returned unchanged.
- A legacy provider without native events falls back to one synthesized event per `SessionInfo`.
- Fallback events copy provider, model, session, title, workdir hash, timestamp, and token usage from the session.
- Missing directories are skipped and operational errors are wrapped with provider context.

## Focused Timezone Tests

Run:

```bash
go test ./cmd -run 'TestResolve(Timezone|DailyDateRange)'
```

This must prove:

- Empty `--timezone` resolves to `time.Local`.
- `Asia/Shanghai` or another valid IANA name resolves successfully.
- Invalid timezone names return an `invalid --timezone` error.
- Explicit `--since` and `--until` dates are parsed in the selected location.
- The default `--days` window starts at local midnight in the selected location, not UTC midnight.

## Broader Gate

After implementation, run:

```bash
go test ./cmd
make fmt
make test
make vet
make build
```

Run `make lint` only if `golangci-lint` is installed.

## OpenSpec Note

Use `event-based-token-aggregation-command-helpers` for EBTA-002. Its final validation task is complete only after owner integration gates pass.
