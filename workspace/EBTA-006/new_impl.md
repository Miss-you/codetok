# EBTA-006 New Implementation Options

## Option A: Reuse `parseUsageCSV` and Convert Rows

Add `CollectUsageEvents(baseDir)` that uses the same path discovery and parser as `CollectSessions`, then maps each `SessionInfo` row to one `UsageEvent`.

Pros:

- Preserves all existing CSV semantics by construction.
- Keeps the change limited to `provider/cursor`.
- Avoids duplicating CSV parsing rules.

Cons:

- Cursor events inherit row-based `SessionID`, which matches current behavior but is not a logical Cursor conversation ID.

## Option B: Add a Separate CSV Event Parser

Parse CSV rows directly into `UsageEvent`.

Pros:

- Can set event fields directly.

Cons:

- Duplicates parsing rules and increases risk of session/event drift.
- More code for no behavior gain.

## Decision

Use Option A. Cursor already treats each CSV row as a session-like local usage record. A native event collector should preserve that contract and only add event-specific metadata (`Timestamp`, `SourcePath`, and `EventID`).
