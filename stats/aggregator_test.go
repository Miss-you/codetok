package stats

import (
	"testing"
	"time"

	"github.com/Miss-you/codetok/provider"
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

	// Results should be sorted by date ascending
	if result[0].Date != "2026-02-15" {
		t.Errorf("result[0].Date = %q, want %q", result[0].Date, "2026-02-15")
	}
	if result[1].Date != "2026-02-17" {
		t.Errorf("result[1].Date = %q, want %q", result[1].Date, "2026-02-17")
	}

	// Day 1: s1 + s3
	if result[0].Sessions != 2 {
		t.Errorf("day1 Sessions = %d, want 2", result[0].Sessions)
	}
	if result[0].TokenUsage.InputOther != 250 {
		t.Errorf("day1 InputOther = %d, want 250", result[0].TokenUsage.InputOther)
	}

	// Day 2: s2 only
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

func TestAggregateByDay_DateFilter(t *testing.T) {
	day1 := time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)

	sessions := []provider.SessionInfo{
		makeSession("s1", day1, 100, 50),
		makeSession("s2", day2, 200, 75),
		makeSession("s3", day3, 300, 100),
	}

	// Filter to only include day2
	since := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	filtered := FilterByDateRange(sessions, since, until)

	if len(filtered) != 1 {
		t.Fatalf("got %d sessions, want 1", len(filtered))
	}
	if filtered[0].SessionID != "s2" {
		t.Errorf("SessionID = %q, want %q", filtered[0].SessionID, "s2")
	}

	// Aggregate the filtered sessions
	result := AggregateByDay(filtered)
	if len(result) != 1 {
		t.Fatalf("got %d days, want 1", len(result))
	}
	if result[0].Date != "2026-02-15" {
		t.Errorf("Date = %q, want %q", result[0].Date, "2026-02-15")
	}

	// Filter with zero since (no lower bound)
	filtered = FilterByDateRange(sessions, time.Time{}, until)
	if len(filtered) != 2 {
		t.Errorf("got %d sessions with no lower bound, want 2", len(filtered))
	}

	// Filter with zero until (no upper bound)
	filtered = FilterByDateRange(sessions, since, time.Time{})
	if len(filtered) != 2 {
		t.Errorf("got %d sessions with no upper bound, want 2", len(filtered))
	}
}
