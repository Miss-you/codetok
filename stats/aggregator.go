package stats

import (
	"sort"
	"time"

	"github.com/miss-you/codetok/provider"
)

// AggregateByDay groups sessions by date and provider, then sums their token usage.
func AggregateByDay(sessions []provider.SessionInfo) []provider.DailyStats {
	if len(sessions) == 0 {
		return nil
	}

	// Key by "date|provider" to group per-provider per-day.
	type dayKey struct {
		date         string
		providerName string
	}
	dayMap := make(map[dayKey]*provider.DailyStats)

	for _, s := range sessions {
		date := "unknown"
		if !s.StartTime.IsZero() {
			date = s.StartTime.Format("2006-01-02")
		}
		key := dayKey{date: date, providerName: s.ProviderName}
		ds, ok := dayMap[key]
		if !ok {
			ds = &provider.DailyStats{Date: date, ProviderName: s.ProviderName}
			dayMap[key] = ds
		}
		ds.Sessions++
		ds.TokenUsage.InputOther += s.TokenUsage.InputOther
		ds.TokenUsage.Output += s.TokenUsage.Output
		ds.TokenUsage.InputCacheRead += s.TokenUsage.InputCacheRead
		ds.TokenUsage.InputCacheCreate += s.TokenUsage.InputCacheCreate
	}

	result := make([]provider.DailyStats, 0, len(dayMap))
	for _, ds := range dayMap {
		result = append(result, *ds)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		return result[i].ProviderName < result[j].ProviderName
	})

	return result
}

// FilterByDateRange returns sessions whose StartTime falls within [since, until].
// A zero-value since or until means no bound on that side.
func FilterByDateRange(sessions []provider.SessionInfo, since, until time.Time) []provider.SessionInfo {
	var filtered []provider.SessionInfo
	for _, s := range sessions {
		if !since.IsZero() && s.StartTime.Before(since) {
			continue
		}
		if !until.IsZero() && s.StartTime.After(until) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}
