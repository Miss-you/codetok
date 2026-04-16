# EBTA-007 Final Implementation v1

## Decision

Implement EBTA-007 by replacing only the `daily` command's data pipeline. The output model remains `[]provider.DailyStats`, so dashboard rendering and JSON response shape do not need a separate migration.

## Code Shape

- `runDaily` delegates to `runDailyWithProviders(cmd, args, provider.Registry(), time.Now())`.
- `runDailyWithProviders` is package-private and exists for deterministic command tests.
- The command collects events with `collectUsageEventsFromProviders`.
- The command reuses `resolveDailyDateRange` for all date flag constraints.
- `dailyEventFilterDates` converts non-zero bounds into localized date keys for `stats.FilterEventsByDateRange`.
- The command aggregates through `stats.AggregateEventsByDayWithDimension`.

## Review Notes

Multi-agent review agreed on the narrow pipeline switch and called out the important risks:

- Keep invalid timezone validation before provider collection.
- Keep JSON ignoring invalid dashboard-only flags.
- Test timezone boundary behavior at the command layer, not only in `stats`.
- Preserve model grouping across providers.

No high-severity design issue was found.
