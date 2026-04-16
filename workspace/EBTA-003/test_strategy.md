# EBTA-003 Test Strategy

## Behaviors to prove

1. `last_token_usage` produces an event from the last-usage payload without subtracting cumulative totals.
2. cumulative `total_token_usage` produces incremental deltas across multiple `token_count` records.
3. emitted events keep the original `token_count` timestamps, including cross-day records.
4. model context falls back to a prior `turn_context.payload.model` when the token record has no model metadata.
5. a decreased cumulative `total_token_usage` is treated as a counter reset, not a negative delta.
6. the first `session_meta.id` and first `user_message` title remain stable even if later metadata appears.
7. `CollectUsageEvents("")` honors `$CODEX_HOME/sessions`.
8. `CollectSessions("")` also honors `$CODEX_HOME/sessions` while existing session parsing stays compatible.
9. an explicit base directory wins over `$CODEX_HOME` for event collection.

## Focused tests

Add tests in `provider/codex/parser_test.go`:

- `TestParseCodexUsageEvents_LastTokenUsageEmitsOneEvent`
- `TestParseCodexUsageEvents_TotalUsageDeltasAcrossDays`
- `TestParseCodexUsageEvents_CumulativeResetStartsFreshDelta`
- `TestParseCodexUsageEvents_KeepsFirstSessionMetadata`
- `TestParseCodexUsageEvents_UsesTurnContextModelFallback`
- `TestCollectCodexUsageEvents_UsesCodexHomeWhenBaseDirEmpty`
- `TestCollectCodexSessions_UsesCodexHomeWhenBaseDirEmpty`
- `TestCollectCodexUsageEvents_ExplicitBaseDirOverridesCodexHome`

Keep the existing `TestParseCodexSession_*` and `TestCollectCodexSessions_DateDirStructure` tests unchanged except for any helper reuse.

## Verification commands

Red phase:

```bash
go test ./provider/codex -run 'Test(ParseCodexUsageEvents|CollectCodexUsageEvents|CodexHome)'
```

Green and package verification:

```bash
go test ./provider/codex -run 'Test(ParseCodexUsageEvents|CollectCodexUsageEvents|CodexHome)'
go test ./provider/codex
```

Repository gates before closing:

```bash
make fmt
make test
make vet
make build
make lint # only if golangci-lint is installed
```
