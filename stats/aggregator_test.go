package stats

import (
	"reflect"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

func makeSession(id string, startTime time.Time, input, output int) provider.SessionInfo {
	return provider.SessionInfo{
		SessionID: id,
		StartTime: startTime,
		EndTime:   startTime.Add(10 * time.Minute),
		Turns:     1,
		TokenUsage: provider.TokenUsage{
			InputOther: input,
			Output:     output,
		},
	}
}

func makeProviderSession(id, providerName string, startTime time.Time, input, output int) provider.SessionInfo {
	s := makeSession(id, startTime, input, output)
	s.ProviderName = providerName
	return s
}

func makeModelSession(id, providerName, modelName string, startTime time.Time, input, output int) provider.SessionInfo {
	s := makeProviderSession(id, providerName, startTime, input, output)
	s.ModelName = modelName
	return s
}

func TestAggregateByDay_SingleDay(t *testing.T) {
	day := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeSession("s1", day, 100, 50),
		makeSession("s2", day.Add(2*time.Hour), 200, 75),
	}

	result := AggregateByDay(sessions)
	if len(result) != 1 {
		t.Fatalf("got %d days, want 1", len(result))
	}
	if result[0].Date != "2026-02-17" {
		t.Errorf("Date = %q, want %q", result[0].Date, "2026-02-17")
	}
	if result[0].GroupBy != string(AggregateDimensionCLI) {
		t.Errorf("GroupBy = %q, want %q", result[0].GroupBy, AggregateDimensionCLI)
	}
	if result[0].Group != "" {
		t.Errorf("Group = %q, want empty", result[0].Group)
	}
	if result[0].ProviderName != "" {
		t.Errorf("ProviderName = %q, want empty", result[0].ProviderName)
	}
	if result[0].Sessions != 2 {
		t.Errorf("Sessions = %d, want 2", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 300 {
		t.Errorf("InputOther = %d, want 300", result[0].TokenUsage.InputOther)
	}
	if result[0].TokenUsage.Output != 125 {
		t.Errorf("Output = %d, want 125", result[0].TokenUsage.Output)
	}
}

func TestAggregateByDay_MultipleDays(t *testing.T) {
	day1 := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 17, 14, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeSession("s1", day1, 100, 50),
		makeSession("s2", day2, 200, 75),
		makeSession("s3", day1.Add(time.Hour), 150, 60),
	}

	result := AggregateByDay(sessions)
	if len(result) != 2 {
		t.Fatalf("got %d days, want 2", len(result))
	}

	if result[0].Date != "2026-02-15" {
		t.Errorf("result[0].Date = %q, want %q", result[0].Date, "2026-02-15")
	}
	if result[1].Date != "2026-02-17" {
		t.Errorf("result[1].Date = %q, want %q", result[1].Date, "2026-02-17")
	}

	if result[0].Sessions != 2 {
		t.Errorf("day1 Sessions = %d, want 2", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 250 {
		t.Errorf("day1 InputOther = %d, want 250", result[0].TokenUsage.InputOther)
	}

	if result[1].Sessions != 1 {
		t.Errorf("day2 Sessions = %d, want 1", result[1].Sessions)
	}
	if result[1].TokenUsage.Output != 75 {
		t.Errorf("day2 Output = %d, want 75", result[1].TokenUsage.Output)
	}
}

func TestAggregateByDay_EmptySessions(t *testing.T) {
	result := AggregateByDay(nil)
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}

	result = AggregateByDay([]provider.SessionInfo{})
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

func TestAggregateByDay_MultipleProvidersSameDay(t *testing.T) {
	day := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeProviderSession("s1", "kimi", day, 100, 50),
		makeProviderSession("s2", "claude", day, 200, 75),
		makeProviderSession("s3", "kimi", day.Add(time.Hour), 150, 60),
		makeProviderSession("s4", "codex", day.Add(2*time.Hour), 300, 100),
	}

	result := AggregateByDay(sessions)
	if len(result) != 3 {
		t.Fatalf("got %d entries, want 3", len(result))
	}

	if result[0].GroupBy != string(AggregateDimensionCLI) || result[0].Group != "claude" || result[0].ProviderName != "claude" {
		t.Errorf("result[0] group meta mismatch: %+v", result[0])
	}
	if result[1].GroupBy != string(AggregateDimensionCLI) || result[1].Group != "codex" || result[1].ProviderName != "codex" {
		t.Errorf("result[1] group meta mismatch: %+v", result[1])
	}
	if result[2].GroupBy != string(AggregateDimensionCLI) || result[2].Group != "kimi" || result[2].ProviderName != "kimi" {
		t.Errorf("result[2] group meta mismatch: %+v", result[2])
	}

	if result[0].Sessions != 1 {
		t.Errorf("claude Sessions = %d, want 1", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 200 {
		t.Errorf("claude InputOther = %d, want 200", result[0].TokenUsage.InputOther)
	}

	if result[1].Sessions != 1 {
		t.Errorf("codex Sessions = %d, want 1", result[1].Sessions)
	}
	if result[1].TokenUsage.Output != 100 {
		t.Errorf("codex Output = %d, want 100", result[1].TokenUsage.Output)
	}

	if result[2].Sessions != 2 {
		t.Errorf("kimi Sessions = %d, want 2", result[2].Sessions)
	}
	if result[2].TokenUsage.InputOther != 250 {
		t.Errorf("kimi InputOther = %d, want 250", result[2].TokenUsage.InputOther)
	}
	if result[2].TokenUsage.Output != 110 {
		t.Errorf("kimi Output = %d, want 110", result[2].TokenUsage.Output)
	}
}

func TestAggregateByDay_DateFilter(t *testing.T) {
	day1 := time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)

	sessions := []provider.SessionInfo{
		makeSession("s1", day1, 100, 50),
		makeSession("s2", day2, 200, 75),
		makeSession("s3", day3, 300, 100),
	}

	since := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	filtered := FilterByDateRange(sessions, since, until)

	if len(filtered) != 1 {
		t.Fatalf("got %d sessions, want 1", len(filtered))
	}
	if filtered[0].SessionID != "s2" {
		t.Errorf("SessionID = %q, want %q", filtered[0].SessionID, "s2")
	}

	result := AggregateByDay(filtered)
	if len(result) != 1 {
		t.Fatalf("got %d days, want 1", len(result))
	}
	if result[0].Date != "2026-02-15" {
		t.Errorf("Date = %q, want %q", result[0].Date, "2026-02-15")
	}

	filtered = FilterByDateRange(sessions, time.Time{}, until)
	if len(filtered) != 2 {
		t.Errorf("got %d sessions with no lower bound, want 2", len(filtered))
	}

	filtered = FilterByDateRange(sessions, since, time.Time{})
	if len(filtered) != 2 {
		t.Errorf("got %d sessions with no upper bound, want 2", len(filtered))
	}
}

func TestAggregateByDayWithDimension_Model(t *testing.T) {
	day := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeModelSession("s1", "claude", "claude-opus-4-6", day, 100, 10),
		makeModelSession("s2", "codex", "gpt-5-codex", day, 200, 20),
		makeModelSession("s3", "claude", "claude-opus-4-6", day.Add(time.Hour), 300, 30),
		makeModelSession("s4", "kimi", "", day.Add(2*time.Hour), 400, 40),
	}

	result := AggregateByDayWithDimension(sessions, AggregateDimensionModel)
	if len(result) != 3 {
		t.Fatalf("got %d entries, want 3", len(result))
	}

	if result[0].Group != "claude-opus-4-6" || result[0].ProviderName != "claude" || result[0].GroupBy != string(AggregateDimensionModel) {
		t.Errorf("result[0] group meta mismatch: %+v", result[0])
	}
	if result[1].Group != "gpt-5-codex" || result[1].ProviderName != "codex" || result[1].GroupBy != string(AggregateDimensionModel) {
		t.Errorf("result[1] group meta mismatch: %+v", result[1])
	}
	if result[2].Group != "unknown (kimi)" || result[2].ProviderName != "kimi" || result[2].GroupBy != string(AggregateDimensionModel) {
		t.Errorf("result[2] group meta mismatch: %+v", result[2])
	}

	if result[0].Sessions != 2 {
		t.Errorf("claude-opus-4-6 Sessions = %d, want 2", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 400 {
		t.Errorf("claude-opus-4-6 InputOther = %d, want 400", result[0].TokenUsage.InputOther)
	}
}

func TestAggregateByDayWithDimension_ModelAcrossProviders(t *testing.T) {
	day := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeModelSession("s1", "claude", "shared-model", day, 100, 10),
		makeModelSession("s2", "codex", "shared-model", day.Add(time.Hour), 200, 20),
		makeModelSession("s3", "claude", "shared-model", day.Add(2*time.Hour), 300, 30),
	}

	result := AggregateByDayWithDimension(sessions, AggregateDimensionModel)
	if len(result) != 1 {
		t.Fatalf("got %d entries, want 1", len(result))
	}
	if result[0].GroupBy != string(AggregateDimensionModel) {
		t.Errorf("GroupBy = %q, want %q", result[0].GroupBy, AggregateDimensionModel)
	}
	if result[0].Group != "shared-model" {
		t.Errorf("Group = %q, want %q", result[0].Group, "shared-model")
	}
	if result[0].ProviderName != "" {
		t.Errorf("ProviderName = %q, want empty for multi-provider group", result[0].ProviderName)
	}
	if !reflect.DeepEqual(result[0].Providers, []string{"claude", "codex"}) {
		t.Errorf("Providers = %v, want [claude codex]", result[0].Providers)
	}
	if result[0].Sessions != 3 {
		t.Errorf("Sessions = %d, want 3", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 600 {
		t.Errorf("InputOther = %d, want 600", result[0].TokenUsage.InputOther)
	}
}

func TestAggregateByDayWithDimension_ModelUnknownGroupedByProvider(t *testing.T) {
	day := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeModelSession("s1", "codex", "", day, 100, 10),
		makeModelSession("s2", "kimi", "   ", day.Add(time.Hour), 200, 20),
		makeModelSession("s3", "codex", "", day.Add(2*time.Hour), 300, 30),
	}

	result := AggregateByDayWithDimension(sessions, AggregateDimensionModel)
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}

	if result[0].Group != "unknown (codex)" {
		t.Errorf("result[0].Group = %q, want %q", result[0].Group, "unknown (codex)")
	}
	if result[0].ProviderName != "codex" {
		t.Errorf("result[0].ProviderName = %q, want %q", result[0].ProviderName, "codex")
	}
	if result[0].Sessions != 2 {
		t.Errorf("result[0].Sessions = %d, want 2", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 400 {
		t.Errorf("result[0].TokenUsage.InputOther = %d, want 400", result[0].TokenUsage.InputOther)
	}

	if result[1].Group != "unknown (kimi)" {
		t.Errorf("result[1].Group = %q, want %q", result[1].Group, "unknown (kimi)")
	}
	if result[1].ProviderName != "kimi" {
		t.Errorf("result[1].ProviderName = %q, want %q", result[1].ProviderName, "kimi")
	}
	if result[1].Sessions != 1 {
		t.Errorf("result[1].Sessions = %d, want 1", result[1].Sessions)
	}
	if result[1].TokenUsage.InputOther != 200 {
		t.Errorf("result[1].TokenUsage.InputOther = %d, want 200", result[1].TokenUsage.InputOther)
	}
}

func TestAggregateByDayWithDimension_InvalidDimensionFallsBackToCLI(t *testing.T) {
	day := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	sessions := []provider.SessionInfo{
		makeModelSession("s1", "claude", "claude-opus-4-6", day, 100, 10),
		makeModelSession("s2", "codex", "gpt-5-codex", day, 200, 20),
	}

	result := AggregateByDayWithDimension(sessions, AggregateDimension("invalid"))
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[0].GroupBy != string(AggregateDimensionCLI) {
		t.Errorf("result[0].GroupBy = %q, want %q", result[0].GroupBy, AggregateDimensionCLI)
	}
	if result[0].Group != "claude" || result[0].ProviderName != "claude" {
		t.Errorf("result[0] group meta mismatch: %+v", result[0])
	}
	if result[1].Group != "codex" || result[1].ProviderName != "codex" {
		t.Errorf("result[1] group meta mismatch: %+v", result[1])
	}
}

func TestNormalizeModelName_Aliases(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		providerName string
		want         string
	}{
		{
			name:         "empty model falls back to provider scoped unknown",
			modelName:    "   ",
			providerName: "kimi",
			want:         "unknown (kimi)",
		},
		{
			name:         "kimi short k2.5 alias",
			modelName:    "K2.5",
			providerName: "kimi",
			want:         "kimi-k2.5",
		},
		{
			name:         "kimi k2 thinking alias",
			modelName:    "k2_thinking",
			providerName: "kimi",
			want:         "kimi-k2-thinking",
		},
		{
			name:         "haiku family short alias",
			modelName:    "Haiku",
			providerName: "claude",
			want:         "claude-haiku",
		},
		{
			name:         "claude 3.5 haiku latest alias",
			modelName:    "Claude-3.5-Haiku-LATEST",
			providerName: "claude",
			want:         "claude-3-5-haiku",
		},
		{
			name:         "claude 3 haiku dated alias",
			modelName:    "claude-3-haiku-20240307",
			providerName: "claude",
			want:         "claude-3-haiku",
		},
		{
			name:         "non alias remains unchanged",
			modelName:    "claude-opus-4-6",
			providerName: "claude",
			want:         "claude-opus-4-6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeModelName(tt.modelName, tt.providerName)
			if got != tt.want {
				t.Fatalf("normalizeModelName(%q, %q) = %q, want %q", tt.modelName, tt.providerName, got, tt.want)
			}
		})
	}
}
