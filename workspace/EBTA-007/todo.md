# EBTA-007 Deferred Items

- EBTA-009 should add binary-level cross-day fixtures for `daily --json` now that command-level event aggregation is in place.
- The daily flag-validation order is intentionally unchanged: local provider collection still happens before `resolveDailyDateRange`. Revisit only if the CLI contract later prioritizes flag-combination errors over local collection errors.
