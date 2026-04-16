# Original Implementation

Kimi currently implements `CollectSessions` only. It discovers session directories under `baseDir/<work-dir-hash>/<session-id>/`, parses `metadata.json`, and scans `wire.jsonl`.

`parseWireJSONL` handles `StatusUpdate` by adding each payload's `token_usage` fields into one session total. It also extracts the first model field seen in a `StatusUpdate` payload for session-level fallback. `TurnBegin` and `TurnEnd` drive session start/end timestamps.

This means the provider discards per-`StatusUpdate` token timestamps before command aggregation can see them.
