# EBTA-006 Final Implementation

Use the `final_impl_v1.md` plan unchanged.

Acceptance checks:

- `provider/cursor.Provider` satisfies `provider.UsageEventProvider`.
- Native Cursor events preserve existing CSV row semantics.
- Invalid local data is skipped the same way as session collection.
- Default root and explicit directory scan behavior are unchanged.
- `cursor/` sync/cache behavior is covered by regression tests, with no code changes expected there.
