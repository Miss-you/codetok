## Why

Antigravity (CLI `agy` and the IDE) keeps every model call's token usage in local SQLite conversation files, but codetok cannot read them, so `daily` and `session` leave Antigravity out. Tracked in issue #53.

## What Changes

- Add an `antigravity` provider that reads `conversations/*.db` under `~/.gemini/antigravity-cli` (CLI) and `~/.gemini/antigravity` (IDE).
- Emit one timestamped `UsageEvent` per model call (`gen_metadata` row) and one `SessionInfo` per conversation.
- Add `--antigravity-dir` to `daily` and `session`; when set, only that root is scanned.
- Decode the needed protobuf fields with a small in-package wire reader. No new dependency (`modernc.org/sqlite` is already used by Cursor activity).

## Capabilities

### New Capabilities

- `antigravity-usage-events`: Antigravity provider emits local token usage events and sessions from conversation databases.

### Modified Capabilities

- None.

## Impact

- New code: `provider/antigravity/`.
- Affected code: blank imports and flags in `cmd/daily.go`, `cmd/session.go`; e2e isolation args; README / README_zh.
- Read-only access to Antigravity files; no remote API calls.
