# EBTA-009 Test Strategy

## Red Test

Add the e2e test first without adding the new cross-day fixture files.

Run:

```bash
go test ./e2e -run TestEventBasedCrossDayAcceptance
```

Expected result: fail because the provider fixture directories are missing or empty, proving the new test exercises new fixture data rather than existing e2e data.

## Green Test

Add the cross-day Codex, Claude, and Kimi fixture files.

Each provider fixture must use the same session identity on both sides of the UTC midnight boundary:

- Codex: one rollout file with one `session_meta.id` and two cumulative `token_count` events.
- Claude: one JSONL file with one `sessionId` and two assistant `message.usage` events.
- Kimi: one `metadata.json` `session_id` and one `wire.jsonl` with two `StatusUpdate.token_usage` events.

The e2e test must run these exact command shapes:

```bash
codetok daily --json --since 2026-04-15 --until 2026-04-16 --timezone UTC
codetok session --json --since 2026-04-16 --until 2026-04-16 --timezone UTC
```

Run:

```bash
go test ./e2e -run TestEventBasedCrossDayAcceptance
```

Expected result: pass with exact daily and session JSON totals:

- daily has 6 rows: 3 providers x 2 dates
- daily 2026-04-15 totals: codex 1300, claude 16, kimi 360
- daily 2026-04-16 totals: codex 650, claude 35, kimi 545
- session filtered to 2026-04-16 has 3 rows with totals 650, 35, and 545

## Broader Verification

After implementation:

```bash
go test ./e2e -run TestEventBasedCrossDayAcceptance
go test ./e2e -run Test
make fmt
make test
make vet
make build
make lint
```

`make lint` is required when `golangci-lint` is installed or the Makefile provides it.
