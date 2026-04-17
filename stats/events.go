package stats

import (
	"sort"
	"strings"
	"time"

	"github.com/miss-you/codetok/provider"
)

type dailyEventKey struct {
	date  string
	group string
}

type dailyEventAggregate struct {
	stats       provider.DailyStats
	providers   map[string]struct{}
	sessionKeys map[string]struct{}
}

// DailyEventAggregator incrementally groups usage events by localized event date and dimension.
type DailyEventAggregator struct {
	dimension AggregateDimension
	loc       *time.Location
	dayMap    map[dailyEventKey]*dailyEventAggregate
}

// NewDailyEventAggregator creates an incremental daily usage event aggregator.
func NewDailyEventAggregator(dimension AggregateDimension, loc *time.Location) *DailyEventAggregator {
	return &DailyEventAggregator{
		dimension: normalizeAggregateDimension(dimension),
		loc:       normalizeEventLocation(loc),
		dayMap:    make(map[dailyEventKey]*dailyEventAggregate),
	}
}

// Add includes one event in the daily aggregate.
func (a *DailyEventAggregator) Add(e provider.UsageEvent) {
	if a.dayMap == nil {
		a.dimension = normalizeAggregateDimension(a.dimension)
		a.loc = normalizeEventLocation(a.loc)
		a.dayMap = make(map[dailyEventKey]*dailyEventAggregate)
	}
	date := e.Timestamp.In(a.loc).Format("2006-01-02")
	group := eventGroupNameForDimension(e, a.dimension)
	key := dailyEventKey{date: date, group: group}
	agg, ok := a.dayMap[key]
	if !ok {
		agg = &dailyEventAggregate{
			stats: provider.DailyStats{
				Date:    date,
				GroupBy: string(a.dimension),
				Group:   group,
			},
			providers:   make(map[string]struct{}),
			sessionKeys: make(map[string]struct{}),
		}
		a.dayMap[key] = agg
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

// Results returns sorted daily stats for all events added so far.
func (a *DailyEventAggregator) Results() []provider.DailyStats {
	if a == nil || len(a.dayMap) == 0 {
		return nil
	}

	result := make([]provider.DailyStats, 0, len(a.dayMap))
	for _, agg := range a.dayMap {
		agg.stats.ProviderName = ""
		agg.stats.Providers = nil
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

// AggregateEventsByDayWithDimension groups usage events by localized event date and dimension.
func AggregateEventsByDayWithDimension(events []provider.UsageEvent, dimension AggregateDimension, loc *time.Location) []provider.DailyStats {
	if len(events) == 0 {
		return nil
	}
	aggregator := NewDailyEventAggregator(dimension, loc)
	for _, e := range events {
		aggregator.Add(e)
	}
	return aggregator.Results()
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
		if EventInDateRange(e, sinceDate, untilDate, loc) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// EventInDateRange reports whether an event's localized date key falls within [sinceDate, untilDate].
// Empty sinceDate or untilDate means no bound on that side.
func EventInDateRange(e provider.UsageEvent, sinceDate, untilDate string, loc *time.Location) bool {
	loc = normalizeEventLocation(loc)
	sinceDate = strings.TrimSpace(sinceDate)
	untilDate = strings.TrimSpace(untilDate)
	date := e.Timestamp.In(loc).Format("2006-01-02")
	if sinceDate != "" && date < sinceDate {
		return false
	}
	if untilDate != "" && date > untilDate {
		return false
	}
	return true
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
