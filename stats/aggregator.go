package stats

import (
	"sort"
	"time"

	"github.com/Miss-you/codetok/provider"
)

// AggregateByDay groups sessions by date and sums their token usage.
func AggregateByDay(sessions []provider.SessionInfo) []provider.DailyStats {
	if len(sessions) == 0 {
		return nil
	}

	dayMap := make(map[string]*provider.DailyStats)

	for _, s := range sessions {
		date := s.StartTime.Format("2006-01-02")
		ds, ok := dayMap[date]
		if !ok {
			ds = &provider.DailyStats{Date: date}
			dayMap[date] = ds
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
		return result[i].Date < result[j].Date
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
