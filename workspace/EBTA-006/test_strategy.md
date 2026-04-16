# EBTA-006 Test Strategy

## Focused Provider Tests

Add tests in `provider/cursor/parser_test.go`:

- `TestCollectUsageEvents_ValidExportMapsRowsToEvents`
  - proves one valid CSV row maps to one `UsageEvent`
  - checks provider, model, session ID, title, timestamp, token fields, source path, and event ID
- `TestCollectUsageEvents_SkipsInvalidCSVFileAndKeepsDeterministicOrder`
  - proves invalid CSV files and malformed rows keep the existing skip behavior
  - proves output order is deterministic
- `TestCollectUsageEvents_DefaultRootAndExplicitDirRules`
  - proves default root scans legacy/imports/synced only
  - proves explicit dir is authoritative
- compile-time assertion that `*Provider` implements `provider.UsageEventProvider`

## Verification Commands

Run in this order:

```bash
go test ./provider/cursor -run 'TestCollectUsageEvents'
go test ./provider/cursor -run 'Test(CollectUsageEvents|ParseUsageCSV|CollectSessions)'
go test ./provider/cursor ./cursor
make fmt
make test
make vet
make build
make lint
```

Run `make lint` only when `golangci-lint` is installed.

## Out Of Scope

- No new stats tests: event aggregation is already covered by EBTA-001.
- No e2e command tests: `daily` and `session` do not consume native events until later EBTA tasks.
- No Cursor sync implementation changes.
