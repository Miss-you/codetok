## 1. Collector Bridge

- [x] 1.1 Add failing command tests for native usage events and legacy session fallback.
- [x] 1.2 Implement `collectUsageEvents` and `collectUsageEventsFromProviders`.
- [x] 1.3 Preserve provider filter, base directory, provider directory override, missing-directory skip, and wrapped operational errors.

## 2. Daily Timezone Helpers

- [x] 2.1 Add failing command tests for empty, valid, and invalid timezone resolution.
- [x] 2.2 Add failing command tests proving daily date windows use the selected location.
- [x] 2.3 Implement `--timezone`, `resolveTimezone`, and location-aware `resolveDailyDateRange`.

## 3. Validation

- [x] 3.1 Run focused collector tests.
- [x] 3.2 Run focused timezone/date-window tests.
- [x] 3.3 Run repository gates for EBTA-002 after owner integration.
