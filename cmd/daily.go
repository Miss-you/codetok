package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
	_ "github.com/miss-you/codetok/provider/claude"
	_ "github.com/miss-you/codetok/provider/codex"
	_ "github.com/miss-you/codetok/provider/kimi"
	"github.com/miss-you/codetok/stats"
)

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Show daily token usage breakdown",
	RunE:  runDaily,
}

const defaultDailyDays = 7
const defaultTokenUnit = "k"

func init() {
	dailyCmd.Flags().Bool("json", false, "Output as JSON")
	dailyCmd.Flags().String("since", "", "Start date filter (format: 2006-01-02)")
	dailyCmd.Flags().String("until", "", "End date filter (format: 2006-01-02)")
	dailyCmd.Flags().Int("days", defaultDailyDays, "Lookback window in days when --since/--until are not set")
	dailyCmd.Flags().Bool("all", false, "Include all historical sessions")
	dailyCmd.Flags().String("unit", defaultTokenUnit, "Token display unit for table output: raw, k, m, g")
	dailyCmd.Flags().String("provider", "", "Filter by provider name (e.g. kimi, claude, codex)")
	dailyCmd.Flags().String("base-dir", "", "Override default data directory (applies to all providers)")
	dailyCmd.Flags().String("kimi-dir", "", "Override Kimi data directory")
	dailyCmd.Flags().String("claude-dir", "", "Override Claude Code data directory")
	dailyCmd.Flags().String("codex-dir", "", "Override Codex CLI data directory")
	rootCmd.AddCommand(dailyCmd)
}

// providerDirFlag returns the per-provider directory override flag name.
func providerDirFlag(name string) string {
	return name + "-dir"
}

func runDaily(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	days, _ := cmd.Flags().GetInt("days")
	allHistory, _ := cmd.Flags().GetBool("all")
	unitStr, _ := cmd.Flags().GetString("unit")
	providerFilter, _ := cmd.Flags().GetString("provider")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	providers := provider.FilterProviders(provider.Registry(), providerFilter)

	var allSessions []provider.SessionInfo
	for _, p := range providers {
		// Check for provider-specific directory override first, then fall back to base-dir
		dir := baseDir
		if providerDir, _ := cmd.Flags().GetString(providerDirFlag(p.Name())); providerDir != "" {
			dir = providerDir
		}
		sessions, err := p.CollectSessions(dir)
		if err != nil {
			// Skip providers whose data directory doesn't exist
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("collecting sessions from %s: %w", p.Name(), err)
		}
		allSessions = append(allSessions, sessions...)
	}

	since, until, err := resolveDailyDateRange(
		sinceStr,
		untilStr,
		days,
		allHistory,
		cmd.Flags().Changed("days"),
		time.Now(),
	)
	if err != nil {
		return err
	}

	allSessions = stats.FilterByDateRange(allSessions, since, until)
	daily := stats.AggregateByDay(allSessions)

	if jsonOutput {
		if daily == nil {
			daily = []provider.DailyStats{}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(daily)
	}

	unit, err := resolveTokenUnit(unitStr)
	if err != nil {
		return err
	}

	printDailyTable(daily, unit)
	return nil
}

type tokenUnit string

const (
	tokenUnitRaw tokenUnit = "raw"
	tokenUnitK   tokenUnit = "k"
	tokenUnitM   tokenUnit = "m"
	tokenUnitG   tokenUnit = "g"
)

func resolveTokenUnit(unit string) (tokenUnit, error) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "raw":
		return tokenUnitRaw, nil
	case "k":
		return tokenUnitK, nil
	case "m":
		return tokenUnitM, nil
	case "g":
		return tokenUnitG, nil
	default:
		return "", fmt.Errorf("invalid --unit: %q (allowed: raw, k, m, g)", unit)
	}
}

func tokenUnitScale(unit tokenUnit) float64 {
	switch unit {
	case tokenUnitK:
		return 1_000
	case tokenUnitM:
		return 1_000_000
	case tokenUnitG:
		return 1_000_000_000
	default:
		return 1
	}
}

func formatTokenByUnit(value int, unit tokenUnit) string {
	if unit == tokenUnitRaw {
		return strconv.Itoa(value)
	}
	scaled := float64(value) / tokenUnitScale(unit)
	return fmt.Sprintf("%.2f%s", scaled, string(unit))
}

func tokenHeader(name string, unit tokenUnit) string {
	if unit == tokenUnitRaw {
		return name
	}
	return fmt.Sprintf("%s(%s)", name, unit)
}

func resolveDailyDateRange(
	sinceStr, untilStr string,
	days int,
	allHistory, daysChanged bool,
	now time.Time,
) (time.Time, time.Time, error) {
	if allHistory {
		if sinceStr != "" || untilStr != "" || daysChanged {
			return time.Time{}, time.Time{}, fmt.Errorf("--all cannot be used with --days, --since, or --until")
		}
		return time.Time{}, time.Time{}, nil
	}
	if days < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --days: must be >= 1")
	}

	var (
		since time.Time
		until time.Time
		err   error
	)
	if sinceStr != "" {
		since, err = time.Parse("2006-01-02", sinceStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --since date: %w", err)
		}
	}
	if untilStr != "" {
		until, err = time.Parse("2006-01-02", untilStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --until date: %w", err)
		}
		// Include the entire "until" day
		until = until.Add(24*time.Hour - time.Nanosecond)
	}
	if sinceStr != "" || untilStr != "" {
		if daysChanged {
			return time.Time{}, time.Time{}, fmt.Errorf("--days cannot be used with --since or --until")
		}
		return since, until, nil
	}

	utcNow := now.UTC()
	startOfToday := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	since = startOfToday.AddDate(0, 0, -(days - 1))
	return since, time.Time{}, nil
}

func printDailyTable(daily []provider.DailyStats, unit tokenUnit) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(
		w,
		"Date\tProvider\tSessions\t%s\t%s\t%s\t%s\t%s\n",
		tokenHeader("Input", unit),
		tokenHeader("Output", unit),
		tokenHeader("Cache Read", unit),
		tokenHeader("Cache Create", unit),
		tokenHeader("Total", unit),
	)

	var totalSessions int
	var totalUsage provider.TokenUsage

	for _, d := range daily {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			d.Date,
			d.ProviderName,
			d.Sessions,
			formatTokenByUnit(d.TokenUsage.InputOther, unit),
			formatTokenByUnit(d.TokenUsage.Output, unit),
			formatTokenByUnit(d.TokenUsage.InputCacheRead, unit),
			formatTokenByUnit(d.TokenUsage.InputCacheCreate, unit),
			formatTokenByUnit(d.TokenUsage.Total(), unit),
		)
		totalSessions += d.Sessions
		totalUsage.InputOther += d.TokenUsage.InputOther
		totalUsage.Output += d.TokenUsage.Output
		totalUsage.InputCacheRead += d.TokenUsage.InputCacheRead
		totalUsage.InputCacheCreate += d.TokenUsage.InputCacheCreate
	}

	fmt.Fprintf(w, "TOTAL\t\t%d\t%s\t%s\t%s\t%s\t%s\n",
		totalSessions,
		formatTokenByUnit(totalUsage.InputOther, unit),
		formatTokenByUnit(totalUsage.Output, unit),
		formatTokenByUnit(totalUsage.InputCacheRead, unit),
		formatTokenByUnit(totalUsage.InputCacheCreate, unit),
		formatTokenByUnit(totalUsage.Total(), unit),
	)

	w.Flush()
}
