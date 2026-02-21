package stats

import (
	"sort"
	"strings"
	"time"

	"github.com/miss-you/codetok/provider"
)

// AggregateDimension defines the grouping dimension used by AggregateByDayWithDimension.
type AggregateDimension string

const (
	// AggregateDimensionCLI groups by provider/CLI name (existing behavior).
	AggregateDimensionCLI AggregateDimension = "cli"
	// AggregateDimensionModel groups by model name.
	AggregateDimensionModel AggregateDimension = "model"
)

// AggregateByDay groups sessions by date and CLI provider (backward-compatible default).
func AggregateByDay(sessions []provider.SessionInfo) []provider.DailyStats {
	return AggregateByDayWithDimension(sessions, AggregateDimensionCLI)
}

// AggregateByDayWithDimension groups sessions by date and the requested dimension.
// DailyStats.ProviderName always keeps provider semantics (CLI/provider name).
// DailyStats.GroupBy and DailyStats.Group carry the explicit grouping result.
func AggregateByDayWithDimension(sessions []provider.SessionInfo, dimension AggregateDimension) []provider.DailyStats {
	if len(sessions) == 0 {
		return nil
	}
	dimension = normalizeAggregateDimension(dimension)

	type dayKey struct {
		date  string
		group string
	}
	type dayAggregate struct {
		stats     provider.DailyStats
		providers map[string]struct{}
	}
	dayMap := make(map[dayKey]*dayAggregate)

	for _, s := range sessions {
		date := "unknown"
		if !s.StartTime.IsZero() {
			date = s.StartTime.Format("2006-01-02")
		}
		group := groupNameForDimension(s, dimension)
		key := dayKey{date: date, group: group}
		agg, ok := dayMap[key]
		if !ok {
			agg = &dayAggregate{
				stats: provider.DailyStats{
					Date:    date,
					GroupBy: string(dimension),
					Group:   group,
				},
				providers: make(map[string]struct{}),
			}
			dayMap[key] = agg
		}
		if providerName := strings.TrimSpace(s.ProviderName); providerName != "" {
			agg.providers[providerName] = struct{}{}
		}
		agg.stats.Sessions++
		agg.stats.TokenUsage.InputOther += s.TokenUsage.InputOther
		agg.stats.TokenUsage.Output += s.TokenUsage.Output
		agg.stats.TokenUsage.InputCacheRead += s.TokenUsage.InputCacheRead
		agg.stats.TokenUsage.InputCacheCreate += s.TokenUsage.InputCacheCreate
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

func normalizeAggregateDimension(dimension AggregateDimension) AggregateDimension {
	switch dimension {
	case AggregateDimensionModel:
		return AggregateDimensionModel
	case AggregateDimensionCLI, "":
		return AggregateDimensionCLI
	default:
		// Keep compatibility for unexpected values by falling back to CLI grouping.
		return AggregateDimensionCLI
	}
}

func groupNameForDimension(s provider.SessionInfo, dimension AggregateDimension) string {
	switch dimension {
	case AggregateDimensionModel:
		return normalizeModelName(s.ModelName, s.ProviderName)
	case AggregateDimensionCLI, "":
		return s.ProviderName
	default:
		// Keep compatibility for unexpected values by falling back to CLI grouping.
		return s.ProviderName
	}
}

func sortedProviderNames(providerSet map[string]struct{}) []string {
	if len(providerSet) == 0 {
		return nil
	}
	names := make([]string, 0, len(providerSet))
	for name := range providerSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeModelName(name, providerName string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		providerName = strings.TrimSpace(providerName)
		if providerName == "" {
			providerName = "unknown"
		}
		return "unknown (" + providerName + ")"
	}
	return normalizeKnownModelAlias(name)
}

func normalizeKnownModelAlias(name string) string {
	alias := strings.ToLower(strings.TrimSpace(name))
	alias = strings.ReplaceAll(alias, "_", "-")
	alias = strings.ReplaceAll(alias, " ", "-")
	for strings.Contains(alias, "--") {
		alias = strings.ReplaceAll(alias, "--", "-")
	}

	switch alias {
	case "k2.5", "k2-5", "kimi-k2.5", "kimi-k2-5":
		return "kimi-k2.5"
	case "k2-thinking", "k2thinking", "kimi-k2-thinking", "kimi-k2thinking":
		return "kimi-k2-thinking"
	case "haiku", "claude-haiku":
		return "claude-haiku"
	}

	if strings.HasPrefix(alias, "claude-3.5-haiku") || strings.HasPrefix(alias, "claude-3-5-haiku") {
		return "claude-3-5-haiku"
	}
	if strings.HasPrefix(alias, "claude-3-haiku") {
		return "claude-3-haiku"
	}
	return name
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
