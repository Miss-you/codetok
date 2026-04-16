package stats

import (
	"sort"
	"strings"
	"time"

	"github.com/miss-you/codetok/provider"
)

// AggregateEventsByDayWithDimension groups usage events by localized event date and dimension.
func AggregateEventsByDayWithDimension(events []provider.UsageEvent, dimension AggregateDimension, loc *time.Location) []provider.DailyStats {
	if len(events) == 0 {
		return nil
	}
	loc = normalizeEventLocation(loc)
	dimension = normalizeAggregateDimension(dimension)

	type dayKey struct {
		date  string
		group string
	}
	type dayAggregate struct {
		stats       provider.DailyStats
		providers   map[string]struct{}
		sessionKeys map[string]struct{}
	}
	dayMap := make(map[dayKey]*dayAggregate)

	for _, e := range events {
		date := e.Timestamp.In(loc).Format("2006-01-02")
		group := eventGroupNameForDimension(e, dimension)
		key := dayKey{date: date, group: group}
		agg, ok := dayMap[key]
		if !ok {
			agg = &dayAggregate{
				stats: provider.DailyStats{
					Date:    date,
					GroupBy: string(dimension),
					Group:   group,
				},
				providers:   make(map[string]struct{}),
				sessionKeys: make(map[string]struct{}),
			}
			dayMap[key] = agg
		}
		if providerName := strings.TrimSpace(e.ProviderName); providerName != "" {
			agg.providers[providerName] = struct{}{}
		}
		sessionKey := eventSessionKey(e)
		if _, ok := agg.sessionKeys[sessionKey]; !ok {
			agg.sessionKeys[sessionKey] = struct{}{}
			agg.stats.Sessions++
		}
		addTokenUsage(&agg.stats.TokenUsage, e.TokenUsage)
	}

	result := make([]provider.DailyStats, 0, len(dayMap))
	for _, agg := range dayMap {
		providers := sortedProviderNames(agg.providers)
		if len(providers) == 1 {
			agg.stats.ProviderName = providers[0]
		} else if len(providers) > 1 {
			agg.stats.Providers = providers
		}
		result = append(result, agg.stats)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		if result[i].ProviderName != result[j].ProviderName {
			return result[i].ProviderName < result[j].ProviderName
		}
		if result[i].GroupBy != result[j].GroupBy {
			return result[i].GroupBy < result[j].GroupBy
		}
		if len(result[i].Providers) != len(result[j].Providers) {
			return len(result[i].Providers) < len(result[j].Providers)
		}
		return strings.Join(result[i].Providers, ",") < strings.Join(result[j].Providers, ",")
	})

	return result
}

// FilterEventsByDateRange returns events whose localized date key falls within [sinceDate, untilDate].
// Empty sinceDate or untilDate means no bound on that side.
func FilterEventsByDateRange(events []provider.UsageEvent, sinceDate, untilDate string, loc *time.Location) []provider.UsageEvent {
	if len(events) == 0 {
		return nil
	}
	loc = normalizeEventLocation(loc)
	sinceDate = strings.TrimSpace(sinceDate)
	untilDate = strings.TrimSpace(untilDate)

	filtered := make([]provider.UsageEvent, 0, len(events))
	for _, e := range events {
		date := e.Timestamp.In(loc).Format("2006-01-02")
		if sinceDate != "" && date < sinceDate {
			continue
		}
		if untilDate != "" && date > untilDate {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func normalizeEventLocation(loc *time.Location) *time.Location {
	if loc == nil {
		return time.Local
	}
	return loc
}

func eventGroupNameForDimension(e provider.UsageEvent, dimension AggregateDimension) string {
	switch dimension {
	case AggregateDimensionModel:
		return normalizeModelName(e.ModelName, e.ProviderName)
	case AggregateDimensionCLI, "":
		return normalizedEventProviderName(e)
	default:
		return normalizedEventProviderName(e)
	}
}

func normalizedEventProviderName(e provider.UsageEvent) string {
	return strings.TrimSpace(e.ProviderName)
}

func eventSessionKey(e provider.UsageEvent) string {
	providerName := normalizedEventProviderName(e)
	if sessionID := strings.TrimSpace(e.SessionID); sessionID != "" {
		return providerName + "\x00session\x00" + sessionID
	}
	if sourcePath := strings.TrimSpace(e.SourcePath); sourcePath != "" {
		return providerName + "\x00source\x00" + sourcePath
	}
	return providerName + "\x00anonymous"
}

func addTokenUsage(dst *provider.TokenUsage, src provider.TokenUsage) {
	dst.InputOther += src.InputOther
	dst.Output += src.Output
	dst.InputCacheRead += src.InputCacheRead
	dst.InputCacheCreate += src.InputCacheCreate
}
