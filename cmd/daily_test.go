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
	now := time.Date(2026, 2, 18, 15, 4, 5, 0, time.FixedZone("UTC+8", 8*3600))

	since, until, err := resolveDailyDateRange("", "", 7, false, false, now)
	if err != nil {
		t.Fatalf("resolveDailyDateRange returned error: %v", err)
	}

	wantSince := time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC)
	if !since.Equal(wantSince) {
		t.Fatalf("since = %v, want %v", since, wantSince)
	}
	if since.Location() != time.UTC {
		t.Fatalf("since location = %v, want UTC", since.Location())
	}
	if !until.IsZero() {
		t.Fatalf("until = %v, want zero time", until)
	}
}

func TestResolveDailyDateRange_WithExplicitDateRange(t *testing.T) {
	since, until, err := resolveDailyDateRange("2026-02-01", "2026-02-15", 7, false, false, time.Now())
	if err != nil {
		t.Fatalf("resolveDailyDateRange returned error: %v", err)
	}

	wantSince := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !since.Equal(wantSince) {
		t.Fatalf("since = %v, want %v", since, wantSince)
	}

	wantUntil := time.Date(2026, 2, 15, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
	if !until.Equal(wantUntil) {
		t.Fatalf("until = %v, want %v", until, wantUntil)
	}
}

func TestResolveDailyDateRange_AllHistory(t *testing.T) {
	since, until, err := resolveDailyDateRange("", "", 7, true, false, time.Now())
	if err != nil {
		t.Fatalf("resolveDailyDateRange returned error: %v", err)
	}
	if !since.IsZero() || !until.IsZero() {
		t.Fatalf("since=%v until=%v, both should be zero", since, until)
	}
}

func TestResolveDailyDateRange_InvalidCombinations(t *testing.T) {
	_, _, err := resolveDailyDateRange("2026-02-01", "", 7, true, false, time.Now())
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
		t.Fatalf("expected --all conflict error, got: %v", err)
	}

	_, _, err = resolveDailyDateRange("2026-02-01", "", 7, false, true, time.Now())
	if err == nil || !strings.Contains(err.Error(), "--days cannot be used") {
		t.Fatalf("expected --days conflict error, got: %v", err)
	}
}

func TestResolveDailyDateRange_AllHistoryConflictPrecedence(t *testing.T) {
	_, _, err := resolveDailyDateRange("", "", 0, true, true, time.Now())
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
		t.Fatalf("expected --all conflict error, got: %v", err)
	}
}

func TestResolveDailyDateRange_InvalidDays(t *testing.T) {
	_, _, err := resolveDailyDateRange("", "", 0, false, false, time.Now())
	if err == nil || !strings.Contains(err.Error(), "invalid --days") {
		t.Fatalf("expected invalid --days error, got: %v", err)
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	output := <-done
	_ = r.Close()
	return output
}

func newDailyTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("until", "", "")
	cmd.Flags().Int("days", defaultDailyDays, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("unit", defaultTokenUnit, "")
	cmd.Flags().String("group-by", defaultGroupBy, "")
	cmd.Flags().Int("top", defaultTopN, "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("base-dir", "", "")
	cmd.Flags().String("kimi-dir", "", "")
	cmd.Flags().String("claude-dir", "", "")
	cmd.Flags().String("codex-dir", "", "")
	return cmd
}
