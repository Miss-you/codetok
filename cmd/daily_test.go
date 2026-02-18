package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
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

	oldStdout := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("opening %s: %v", os.DevNull, err)
	}
	os.Stdout = devNull
	defer func() {
		os.Stdout = oldStdout
		_ = devNull.Close()
	}()

	if err := runDaily(cmd, nil); err != nil {
		t.Fatalf("runDaily returned error for json output with invalid unit: %v", err)
	}
}

func newDailyTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("until", "", "")
	cmd.Flags().Int("days", defaultDailyDays, "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("unit", defaultTokenUnit, "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("base-dir", "", "")
	return cmd
}
