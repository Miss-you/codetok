package stats

import (
	"reflect"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

func makeUsageEvent(sessionID, providerName, modelName string, timestamp time.Time, input, output int) provider.UsageEvent {
	return provider.UsageEvent{
		ProviderName: providerName,
		ModelName:    modelName,
		SessionID:    sessionID,
		Timestamp:    timestamp,
		TokenUsage: provider.TokenUsage{
			InputOther: input,
			Output:     output,
		},
	}
}

func TestAggregateEventsByDayWithDimension_SplitsSameSessionAcrossDays(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	events := []provider.UsageEvent{
		makeUsageEvent("same-session", "codex", "gpt-5.4", time.Date(2026, 4, 15, 23, 50, 0, 0, loc), 100, 10),
		makeUsageEvent("same-session", "codex", "gpt-5.4", time.Date(2026, 4, 16, 0, 10, 0, 0, loc), 200, 20),
	}

	got := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, loc)

	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %#v", len(got), got)
	}
	if got[0].Date != "2026-04-15" || got[0].TokenUsage.Total() != 110 {
		t.Fatalf("first row mismatch: %#v", got[0])
	}
	if got[0].Sessions != 1 || got[0].ProviderName != "codex" || got[0].Group != "codex" {
		t.Fatalf("first row metadata mismatch: %#v", got[0])
	}
	if got[1].Date != "2026-04-16" || got[1].TokenUsage.Total() != 220 {
		t.Fatalf("second row mismatch: %#v", got[1])
	}
	if got[1].Sessions != 1 || got[1].ProviderName != "codex" || got[1].Group != "codex" {
		t.Fatalf("second row metadata mismatch: %#v", got[1])
	}
}

func TestAggregateEventsByDayWithDimension_UsesRequestedTimezone(t *testing.T) {
	utc := time.UTC
	shanghai := time.FixedZone("UTC+8", 8*3600)
	events := []provider.UsageEvent{
		makeUsageEvent("s1", "codex", "", time.Date(2026, 4, 15, 18, 0, 0, 0, utc), 1, 0),
	}

	gotUTC := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, utc)
	gotShanghai := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, shanghai)

	if gotUTC[0].Date != "2026-04-15" {
		t.Fatalf("UTC date = %q, want 2026-04-15", gotUTC[0].Date)
	}
	if gotShanghai[0].Date != "2026-04-16" {
		t.Fatalf("Shanghai date = %q, want 2026-04-16", gotShanghai[0].Date)
	}
}

func TestAggregateEventsByDayWithDimension_CountsDistinctSessionsNotEvents(t *testing.T) {
	day := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	events := []provider.UsageEvent{
		makeUsageEvent("s1", "codex", "", day, 100, 10),
		makeUsageEvent("s1", "codex", "", day.Add(time.Hour), 200, 20),
		makeUsageEvent("s2", "codex", "", day.Add(2*time.Hour), 300, 30),
	}

	got := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, time.UTC)

	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %#v", len(got), got)
	}
	if got[0].Sessions != 2 {
		t.Fatalf("Sessions = %d, want 2", got[0].Sessions)
	}
	if got[0].TokenUsage.Total() != 660 {
		t.Fatalf("Total = %d, want 660", got[0].TokenUsage.Total())
	}
}

func TestAggregateEventsByDayWithDimension_TrimsCLIProviderGroups(t *testing.T) {
	day := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	events := []provider.UsageEvent{
		makeUsageEvent("s1", " codex ", "", day, 100, 10),
		makeUsageEvent("s2", "codex", "", day.Add(time.Hour), 200, 20),
	}

	got := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, time.UTC)

	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %#v", len(got), got)
	}
	if got[0].Group != "codex" || got[0].ProviderName != "codex" {
		t.Fatalf("group metadata mismatch: %#v", got[0])
	}
	if got[0].Sessions != 2 {
		t.Fatalf("Sessions = %d, want 2", got[0].Sessions)
	}
	if got[0].TokenUsage.Total() != 330 {
		t.Fatalf("Total = %d, want 330", got[0].TokenUsage.Total())
	}
}

func TestAggregateEventsByDayWithDimension_UsesSourcePathFallbackForSessionCount(t *testing.T) {
	day := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	events := []provider.UsageEvent{
		{
			ProviderName: "codex",
			Timestamp:    day,
			SourcePath:   "rollout-a.jsonl",
			EventID:      "event-1",
			TokenUsage:   provider.TokenUsage{InputOther: 10},
		},
		{
			ProviderName: "codex",
			Timestamp:    day.Add(time.Hour),
			SourcePath:   "rollout-a.jsonl",
			EventID:      "event-2",
			TokenUsage:   provider.TokenUsage{InputOther: 20},
		},
		{
			ProviderName: "codex",
			Timestamp:    day.Add(2 * time.Hour),
			SourcePath:   "rollout-b.jsonl",
			EventID:      "event-3",
			TokenUsage:   provider.TokenUsage{InputOther: 30},
		},
	}

	got := AggregateEventsByDayWithDimension(events, AggregateDimensionCLI, time.UTC)

	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %#v", len(got), got)
	}
	if got[0].Sessions != 2 {
		t.Fatalf("Sessions = %d, want 2", got[0].Sessions)
	}
	if got[0].TokenUsage.InputOther != 60 {
		t.Fatalf("InputOther = %d, want 60", got[0].TokenUsage.InputOther)
	}
}

func TestAggregateEventsByDayWithDimension_ModelAcrossProviders(t *testing.T) {
	day := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	events := []provider.UsageEvent{
		makeUsageEvent("s1", "claude", "shared-model", day, 100, 10),
		makeUsageEvent("s2", "codex", "shared-model", day.Add(time.Hour), 200, 20),
		makeUsageEvent("s3", "claude", "shared-model", day.Add(2*time.Hour), 300, 30),
	}

	got := AggregateEventsByDayWithDimension(events, AggregateDimensionModel, time.UTC)

	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %#v", len(got), got)
	}
	if got[0].GroupBy != string(AggregateDimensionModel) || got[0].Group != "shared-model" {
		t.Fatalf("group metadata mismatch: %#v", got[0])
	}
	if got[0].ProviderName != "" {
		t.Fatalf("ProviderName = %q, want empty for multi-provider group", got[0].ProviderName)
	}
	if !reflect.DeepEqual(got[0].Providers, []string{"claude", "codex"}) {
		t.Fatalf("Providers = %v, want [claude codex]", got[0].Providers)
	}
	if got[0].Sessions != 3 {
		t.Fatalf("Sessions = %d, want 3", got[0].Sessions)
	}
}

func TestFilterEventsByDateRange_UsesLocalizedInclusiveDateKeys(t *testing.T) {
	utc := time.UTC
	shanghai := time.FixedZone("UTC+8", 8*3600)
	events := []provider.UsageEvent{
		makeUsageEvent("before", "codex", "", time.Date(2026, 4, 15, 15, 0, 0, 0, utc), 1, 0),
		makeUsageEvent("inside-1", "codex", "", time.Date(2026, 4, 15, 16, 0, 0, 0, utc), 2, 0),
		makeUsageEvent("inside-2", "codex", "", time.Date(2026, 4, 16, 15, 59, 59, 0, utc), 3, 0),
		makeUsageEvent("after", "codex", "", time.Date(2026, 4, 16, 16, 0, 0, 0, utc), 4, 0),
	}

	got := FilterEventsByDateRange(events, "2026-04-16", "2026-04-16", shanghai)

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(got), got)
	}
	if got[0].SessionID != "inside-1" || got[1].SessionID != "inside-2" {
		t.Fatalf("filtered sessions = [%s %s], want [inside-1 inside-2]", got[0].SessionID, got[1].SessionID)
	}
}
