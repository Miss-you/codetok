package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
	"github.com/miss-you/codetok/stats"
)

func TestResolveDailyDateRange_DefaultWindow(t *testing.T) {
	loc := mustLoadLocation(t, "Asia/Shanghai")
	now := time.Date(2026, 2, 18, 15, 4, 5, 0, time.UTC)

	since, until, err := resolveDailyDateRange("", "", 7, false, false, now, loc)
	if err != nil {
		t.Fatalf("resolveDailyDateRange returned error: %v", err)
	}

	wantSince := time.Date(2026, 2, 12, 0, 0, 0, 0, loc)
	if !since.Equal(wantSince) {
		t.Fatalf("since = %v, want %v", since, wantSince)
	}
	if since.Location() != loc {
		t.Fatalf("since location = %v, want %v", since.Location(), loc)
	}
	if !until.IsZero() {
		t.Fatalf("until = %v, want zero time", until)
	}
}

func TestResolveDailyDateRange_WithExplicitDateRange(t *testing.T) {
	loc := mustLoadLocation(t, "Asia/Shanghai")

	since, until, err := resolveDailyDateRange("2026-02-01", "2026-02-15", 7, false, false, time.Now(), loc)
	if err != nil {
		t.Fatalf("resolveDailyDateRange returned error: %v", err)
	}

	wantSince := time.Date(2026, 2, 1, 0, 0, 0, 0, loc)
	if !since.Equal(wantSince) {
		t.Fatalf("since = %v, want %v", since, wantSince)
	}
	if since.Location() != loc {
		t.Fatalf("since location = %v, want %v", since.Location(), loc)
	}

	wantUntil := time.Date(2026, 2, 15, 23, 59, 59, int(time.Second-time.Nanosecond), loc)
	if !until.Equal(wantUntil) {
		t.Fatalf("until = %v, want %v", until, wantUntil)
	}
	if until.Location() != loc {
		t.Fatalf("until location = %v, want %v", until.Location(), loc)
	}
}

func TestResolveDailyDateRange_AllHistory(t *testing.T) {
	since, until, err := resolveDailyDateRange("", "", 7, true, false, time.Now(), time.UTC)
	if err != nil {
		t.Fatalf("resolveDailyDateRange returned error: %v", err)
	}
	if !since.IsZero() || !until.IsZero() {
		t.Fatalf("since=%v until=%v, both should be zero", since, until)
	}
}

func TestResolveDailyDateRange_InvalidCombinations(t *testing.T) {
	_, _, err := resolveDailyDateRange("2026-02-01", "", 7, true, false, time.Now(), time.UTC)
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
		t.Fatalf("expected --all conflict error, got: %v", err)
	}

	_, _, err = resolveDailyDateRange("2026-02-01", "", 7, false, true, time.Now(), time.UTC)
	if err == nil || !strings.Contains(err.Error(), "--days cannot be used") {
		t.Fatalf("expected --days conflict error, got: %v", err)
	}
}

func TestResolveDailyDateRange_AllHistoryConflictPrecedence(t *testing.T) {
	_, _, err := resolveDailyDateRange("", "", 0, true, true, time.Now(), time.UTC)
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
		t.Fatalf("expected --all conflict error, got: %v", err)
	}
}

func TestResolveDailyDateRange_InvalidDays(t *testing.T) {
	_, _, err := resolveDailyDateRange("", "", 0, false, false, time.Now(), time.UTC)
	if err == nil || !strings.Contains(err.Error(), "invalid --days") {
		t.Fatalf("expected invalid --days error, got: %v", err)
	}
}

func TestResolveTimezone(t *testing.T) {
	got, err := resolveTimezone("")
	if err != nil {
		t.Fatalf("resolveTimezone(\"\") returned error: %v", err)
	}
	if got != time.Local {
		t.Fatalf("resolveTimezone(\"\") = %v, want time.Local", got)
	}

	got, err = resolveTimezone("Asia/Shanghai")
	if err != nil {
		t.Fatalf("resolveTimezone(\"Asia/Shanghai\") returned error: %v", err)
	}
	if got.String() != "Asia/Shanghai" {
		t.Fatalf("resolveTimezone(\"Asia/Shanghai\") = %v, want Asia/Shanghai", got)
	}
}

func TestResolveTimezone_Invalid(t *testing.T) {
	_, err := resolveTimezone("not/a-zone")
	if err == nil || !strings.Contains(err.Error(), "invalid --timezone") {
		t.Fatalf("expected invalid --timezone error, got: %v", err)
	}
}

func TestResolveTimezone_RunDailyRejectsInvalidFlag(t *testing.T) {
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("timezone", "not/a-zone"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}
	if err := cmd.Flags().Set("provider", "nonexistent"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	err := runDaily(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid --timezone") {
		t.Fatalf("expected invalid --timezone error, got: %v", err)
	}
}

func TestResolveTokenUnit(t *testing.T) {
	tests := []struct {
		input string
		want  tokenUnit
	}{
		{input: "raw", want: tokenUnitRaw},
		{input: "k", want: tokenUnitK},
		{input: "m", want: tokenUnitM},
		{input: "g", want: tokenUnitG},
		{input: "K", want: tokenUnitK},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := resolveTokenUnit(tt.input)
			if err != nil {
				t.Fatalf("resolveTokenUnit(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("resolveTokenUnit(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTokenUnit_Invalid(t *testing.T) {
	_, err := resolveTokenUnit("x")
	if err == nil {
		t.Fatal("expected error for invalid unit, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --unit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveGroupBy(t *testing.T) {
	tests := []struct {
		input string
		want  stats.AggregateDimension
	}{
		{input: "model", want: stats.AggregateDimensionModel},
		{input: "MODEL", want: stats.AggregateDimensionModel},
		{input: "cli", want: stats.AggregateDimensionCLI},
		{input: "", want: stats.AggregateDimensionCLI},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := resolveGroupBy(tt.input)
			if err != nil {
				t.Fatalf("resolveGroupBy(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("resolveGroupBy(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveGroupBy_Invalid(t *testing.T) {
	_, err := resolveGroupBy("provider")
	if err == nil {
		t.Fatal("expected error for invalid group-by, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --group-by") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatTokenByUnit(t *testing.T) {
	tests := []struct {
		value int
		unit  tokenUnit
		want  string
	}{
		{value: 1234, unit: tokenUnitRaw, want: "1234"},
		{value: 1234, unit: tokenUnitK, want: "1.23k"},
		{value: 1234567, unit: tokenUnitM, want: "1.23m"},
		{value: 1500000000, unit: tokenUnitG, want: "1.50g"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_%s", tt.value, tt.unit), func(t *testing.T) {
			got := formatTokenByUnit(tt.value, tt.unit)
			if got != tt.want {
				t.Fatalf("formatTokenByUnit(%d, %q) = %q, want %q", tt.value, tt.unit, got, tt.want)
			}
		})
	}
}

func TestRunDaily_JSONIgnoresInvalidUnit(t *testing.T) {
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("unit", "invalid"); err != nil {
		t.Fatalf("setting --unit: %v", err)
	}
	if err := cmd.Flags().Set("provider", "nonexistent"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}
	if err := cmd.Flags().Set("group-by", "cli"); err != nil {
		t.Fatalf("setting --group-by: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runDaily(cmd, nil); err != nil {
			t.Fatalf("runDaily returned error for json output with invalid unit: %v", err)
		}
	})

	var got []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json output should be parseable: %v\noutput: %s", err, output)
	}
	if strings.Contains(output, "Daily Total Trend") {
		t.Fatalf("json output polluted by UI text:\n%s", output)
	}
}

func TestRunDaily_InvalidTop(t *testing.T) {
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("top", "0"); err != nil {
		t.Fatalf("setting --top: %v", err)
	}
	if err := cmd.Flags().Set("provider", "nonexistent"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	err := runDaily(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid --top") {
		t.Fatalf("expected invalid --top error, got: %v", err)
	}
}

func TestRunDaily_JSONIgnoresInvalidTop(t *testing.T) {
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("top", "0"); err != nil {
		t.Fatalf("setting --top: %v", err)
	}
	if err := cmd.Flags().Set("provider", "nonexistent"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runDaily(cmd, nil); err != nil {
			t.Fatalf("runDaily returned error for json output with invalid top: %v", err)
		}
	})

	var got []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json output should be parseable: %v\noutput: %s", err, output)
	}
	if strings.Contains(output, "Daily Total Trend") {
		t.Fatalf("json output polluted by UI text:\n%s", output)
	}
}

func TestRunDaily_JSONAggregatesUsageEventsByEventDate(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{
			{
				ProviderName: "codex",
				ModelName:    "gpt-5.4",
				SessionID:    "same-session",
				Timestamp:    time.Date(2026, 4, 15, 23, 50, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 100, Output: 10},
			},
			{
				ProviderName: "codex",
				ModelName:    "gpt-5.4",
				SessionID:    "same-session",
				Timestamp:    time.Date(2026, 4, 16, 0, 10, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 200, Output: 20},
			},
		},
	}
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("setting --all: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "UTC"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runDailyWithProviders(cmd, nil, []provider.Provider{eventProvider}, time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)); err != nil {
			t.Fatalf("runDailyWithProviders returned error: %v", err)
		}
	})

	got := decodeDailyJSON(t, output)
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %#v", len(got), got)
	}
	if got[0].Date != "2026-04-15" || got[0].Sessions != 1 || got[0].TokenUsage.Total() != 110 {
		t.Fatalf("first row mismatch: %#v", got[0])
	}
	if got[0].ProviderName != "codex" || got[0].GroupBy != "cli" || got[0].Group != "codex" {
		t.Fatalf("first row grouping mismatch: %#v", got[0])
	}
	if got[1].Date != "2026-04-16" || got[1].Sessions != 1 || got[1].TokenUsage.Total() != 220 {
		t.Fatalf("second row mismatch: %#v", got[1])
	}
}

func TestRunDaily_JSONTimezoneChangesEventDateKeys(t *testing.T) {
	event := provider.UsageEvent{
		ProviderName: "codex",
		ModelName:    "gpt-5.4",
		SessionID:    "timezone-session",
		Timestamp:    time.Date(2026, 4, 15, 18, 0, 0, 0, time.UTC),
		TokenUsage:   provider.TokenUsage{InputOther: 1},
	}

	run := func(timezone string) []provider.DailyStats {
		cmd := newDailyTestCommand()
		if err := cmd.Flags().Set("json", "true"); err != nil {
			t.Fatalf("setting --json: %v", err)
		}
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("setting --all: %v", err)
		}
		if err := cmd.Flags().Set("timezone", timezone); err != nil {
			t.Fatalf("setting --timezone: %v", err)
		}
		eventProvider := &collectTestUsageEventProvider{
			collectTestProvider: collectTestProvider{name: "codex"},
			events:              []provider.UsageEvent{event},
		}

		output := captureStdout(t, func() {
			if err := runDailyWithProviders(cmd, nil, []provider.Provider{eventProvider}, time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)); err != nil {
				t.Fatalf("runDailyWithProviders returned error: %v", err)
			}
		})
		return decodeDailyJSON(t, output)
	}

	gotUTC := run("UTC")
	gotShanghai := run("Asia/Shanghai")

	if len(gotUTC) != 1 || gotUTC[0].Date != "2026-04-15" {
		t.Fatalf("UTC rows = %#v, want one 2026-04-15 row", gotUTC)
	}
	if len(gotShanghai) != 1 || gotShanghai[0].Date != "2026-04-16" {
		t.Fatalf("Shanghai rows = %#v, want one 2026-04-16 row", gotShanghai)
	}
}

func TestRunDaily_DefaultWindowFiltersByLocalEventDate(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{
			{
				ProviderName: "codex",
				SessionID:    "outside-local-window",
				Timestamp:    time.Date(2026, 4, 9, 15, 59, 59, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 100},
			},
			{
				ProviderName: "codex",
				SessionID:    "inside-local-window",
				Timestamp:    time.Date(2026, 4, 9, 16, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 200},
			},
		},
	}
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "Asia/Shanghai"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}
	now := time.Date(2026, 4, 16, 1, 0, 0, 0, mustLoadLocation(t, "Asia/Shanghai"))

	output := captureStdout(t, func() {
		if err := runDailyWithProviders(cmd, nil, []provider.Provider{eventProvider}, now); err != nil {
			t.Fatalf("runDailyWithProviders returned error: %v", err)
		}
	})

	got := decodeDailyJSON(t, output)
	if len(got) != 1 {
		t.Fatalf("got %d rows, want one in-window row: %#v", len(got), got)
	}
	if got[0].Date != "2026-04-10" {
		t.Fatalf("row = %#v, want 2026-04-10 aggregate row", got[0])
	}
	if got[0].TokenUsage.Total() != 200 {
		t.Fatalf("total = %d, want only inside event total 200", got[0].TokenUsage.Total())
	}
}

func TestRunDaily_ExplicitDateRangeFiltersByLocalEventDate(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{
			{
				ProviderName: "codex",
				SessionID:    "before-local-day",
				Timestamp:    time.Date(2026, 4, 15, 15, 59, 59, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 100},
			},
			{
				ProviderName: "codex",
				SessionID:    "start-local-day",
				Timestamp:    time.Date(2026, 4, 15, 16, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 200},
			},
			{
				ProviderName: "codex",
				SessionID:    "end-local-day",
				Timestamp:    time.Date(2026, 4, 16, 15, 59, 59, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 300},
			},
			{
				ProviderName: "codex",
				SessionID:    "after-local-day",
				Timestamp:    time.Date(2026, 4, 16, 16, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 400},
			},
		},
	}
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("since", "2026-04-16"); err != nil {
		t.Fatalf("setting --since: %v", err)
	}
	if err := cmd.Flags().Set("until", "2026-04-16"); err != nil {
		t.Fatalf("setting --until: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "Asia/Shanghai"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runDailyWithProviders(cmd, nil, []provider.Provider{eventProvider}, time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)); err != nil {
			t.Fatalf("runDailyWithProviders returned error: %v", err)
		}
	})

	got := decodeDailyJSON(t, output)
	if len(got) != 1 {
		t.Fatalf("got %d rows, want one 2026-04-16 row: %#v", len(got), got)
	}
	if got[0].Date != "2026-04-16" || got[0].TokenUsage.Total() != 500 || got[0].Sessions != 2 {
		t.Fatalf("row = %#v, want only two events on Shanghai 2026-04-16", got[0])
	}
}

func TestRunDaily_JSONModelGroupingUsesUsageEventsAcrossProviders(t *testing.T) {
	codexProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{{
			ProviderName: "codex",
			ModelName:    "shared-model",
			SessionID:    "codex-session",
			Timestamp:    time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 25},
		}},
	}
	claudeProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "claude"},
		events: []provider.UsageEvent{{
			ProviderName: "claude",
			ModelName:    "shared-model",
			SessionID:    "claude-session",
			Timestamp:    time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 75},
		}},
	}
	cmd := newDailyTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatalf("setting --all: %v", err)
	}
	if err := cmd.Flags().Set("group-by", "model"); err != nil {
		t.Fatalf("setting --group-by: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runDailyWithProviders(cmd, nil, []provider.Provider{codexProvider, claudeProvider}, time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)); err != nil {
			t.Fatalf("runDailyWithProviders returned error: %v", err)
		}
	})

	got := decodeDailyJSON(t, output)
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %#v", len(got), got)
	}
	if got[0].ProviderName != "" || got[0].GroupBy != "model" || got[0].Group != "shared-model" {
		t.Fatalf("grouping mismatch: %#v", got[0])
	}
	if strings.Join(got[0].Providers, ",") != "claude,codex" {
		t.Fatalf("providers = %#v, want [claude codex]", got[0].Providers)
	}
	if got[0].TokenUsage.Total() != 100 || got[0].Sessions != 2 {
		t.Fatalf("aggregate mismatch: %#v", got[0])
	}
}

func TestPrintDailyDashboard_ThreeSectionLayout_Model(t *testing.T) {
	daily := []provider.DailyStats{
		{
			Date:         "2026-02-14",
			ProviderName: "gpt-5-codex",
			Sessions:     1,
			TokenUsage: provider.TokenUsage{
				InputOther:       100,
				Output:           50,
				InputCacheRead:   200,
				InputCacheCreate: 10,
			},
		},
		{
			Date:         "2026-02-15",
			ProviderName: "gpt-5-codex",
			Sessions:     1,
			TokenUsage: provider.TokenUsage{
				InputOther:       200,
				Output:           80,
				InputCacheRead:   100,
				InputCacheCreate: 20,
			},
		},
		{
			Date:         "2026-02-15",
			ProviderName: "claude-opus-4-6",
			Sessions:     2,
			TokenUsage: provider.TokenUsage{
				InputOther:       400,
				Output:           100,
				InputCacheRead:   200,
				InputCacheCreate: 40,
			},
		},
	}

	output := captureStdout(t, func() {
		printDailyDashboard(daily, tokenUnitK, stats.AggregateDimensionModel, 2)
	})

	assertContainsAll(t, output,
		"Daily Total Trend",
		"Model Total Ranking",
		"Top 2 Model Share",
		"Coverage:",
		"Bar",
		"gpt-5-codex",
		"claude-opus-4-6",
	)
}

func TestPrintDailyDashboard_GroupByCLI(t *testing.T) {
	daily := []provider.DailyStats{
		{
			Date:         "2026-02-14",
			ProviderName: "kimi",
			Sessions:     1,
			TokenUsage: provider.TokenUsage{
				InputOther:       500,
				Output:           250,
				InputCacheRead:   1000,
				InputCacheCreate: 50,
			},
		},
		{
			Date:         "2026-02-15",
			ProviderName: "codex",
			Sessions:     1,
			TokenUsage: provider.TokenUsage{
				InputOther:       100,
				Output:           50,
				InputCacheRead:   200,
				InputCacheCreate: 10,
			},
		},
	}

	output := captureStdout(t, func() {
		printDailyDashboard(daily, tokenUnitRaw, stats.AggregateDimensionCLI, 1)
	})

	assertContainsAll(t, output,
		"CLI Total Ranking",
		"Top 1 CLI Share",
		"kimi",
	)
}

func TestPrintTopGroupShare_RespectsTopN(t *testing.T) {
	groupTotals := []groupTotal{
		{
			Name:     "kimi",
			Sessions: 3,
			TokenUsage: provider.TokenUsage{
				InputOther:       900,
				Output:           300,
				InputCacheRead:   600,
				InputCacheCreate: 150,
			},
		},
		{
			Name:     "codex",
			Sessions: 2,
			TokenUsage: provider.TokenUsage{
				InputOther:       200,
				Output:           100,
				InputCacheRead:   100,
				InputCacheCreate: 50,
			},
		},
	}

	output := captureStdout(t, func() {
		printTopGroupShare(groupTotals, tokenUnitRaw, stats.AggregateDimensionCLI, 1)
	})

	assertContainsAll(t, output, "Top 1 CLI Share", "Coverage:")
	if strings.Contains(output, "\n2\t") {
		t.Fatalf("top=1 should only print one ranked row:\n%s", output)
	}
	if strings.Contains(output, "codex\t") {
		t.Fatalf("top=1 should not include second group in share table:\n%s", output)
	}
}

func assertContainsAll(t *testing.T, text string, values ...string) {
	t.Helper()
	for _, v := range values {
		if !strings.Contains(text, v) {
			t.Fatalf("output missing %q:\n%s", v, text)
		}
	}
}

func captureStdout(t *testing.T, fn func()) (output string) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	defer func() {
		_ = w.Close()
		os.Stdout = oldStdout
		output = <-done
		_ = r.Close()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	return output
}

func decodeDailyJSON(t *testing.T, output string) []provider.DailyStats {
	t.Helper()
	var got []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decoding daily json: %v\noutput: %s", err, output)
	}
	return got
}

func newDailyTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("until", "", "")
	cmd.Flags().Int("days", defaultDailyDays, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("timezone", "", "")
	cmd.Flags().String("unit", defaultTokenUnit, "")
	cmd.Flags().String("group-by", defaultGroupBy, "")
	cmd.Flags().Int("top", defaultTopN, "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("base-dir", "", "")
	cmd.Flags().String("kimi-dir", "", "")
	cmd.Flags().String("claude-dir", "", "")
	cmd.Flags().String("codex-dir", "", "")
	cmd.Flags().String("cursor-dir", "", "")
	return cmd
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("loading location %q: %v", name, err)
	}
	return loc
}
