package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
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
const defaultTokenUnit = "m"
const defaultGroupBy = "cli"
const defaultTopN = 5

func init() {
	dailyCmd.Flags().Bool("json", false, "Output as JSON")
	dailyCmd.Flags().String("since", "", "Start date filter (format: 2006-01-02)")
	dailyCmd.Flags().String("until", "", "End date filter (format: 2006-01-02)")
	dailyCmd.Flags().Int("days", defaultDailyDays, "Lookback window in days when --since/--until are not set")
	dailyCmd.Flags().Bool("all", false, "Include all historical sessions")
	dailyCmd.Flags().String("unit", defaultTokenUnit, "Token display unit for dashboard output: raw, k, m, g")
	dailyCmd.Flags().String("group-by", defaultGroupBy, "Group by dimension for aggregation: cli, model")
	dailyCmd.Flags().Int("top", defaultTopN, "Top N groups to show in dashboard share section")
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
	groupByStr, _ := cmd.Flags().GetString("group-by")
	topN, _ := cmd.Flags().GetInt("top")
	providerFilter, _ := cmd.Flags().GetString("provider")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	groupBy, err := resolveGroupBy(groupByStr)
	if err != nil {
		return err
	}
	if !jsonOutput && topN < 1 {
		return fmt.Errorf("invalid --top: must be >= 1")
	}

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
	daily := stats.AggregateByDayWithDimension(allSessions, groupBy)

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

	printDailyDashboard(daily, unit, groupBy, topN)
	return nil
}

func resolveGroupBy(groupBy string) (stats.AggregateDimension, error) {
	switch strings.ToLower(strings.TrimSpace(groupBy)) {
	case "":
		return stats.AggregateDimensionCLI, nil
	case "model":
		return stats.AggregateDimensionModel, nil
	case "cli":
		return stats.AggregateDimensionCLI, nil
	default:
		return "", fmt.Errorf("invalid --group-by: %q (allowed: model, cli)", groupBy)
	}
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

type dayTotal struct {
	Date       string
	Sessions   int
	TokenUsage provider.TokenUsage
}

type groupTotal struct {
	Name       string
	Sessions   int
	TokenUsage provider.TokenUsage
}

func aggregateTotalsByDate(daily []provider.DailyStats) []dayTotal {
	if len(daily) == 0 {
		return nil
	}

	dayMap := make(map[string]*dayTotal)
	for _, d := range daily {
		t, ok := dayMap[d.Date]
		if !ok {
			t = &dayTotal{Date: d.Date}
			dayMap[d.Date] = t
		}
		t.Sessions += d.Sessions
		mergeTokenUsage(&t.TokenUsage, d.TokenUsage)
	}

	result := make([]dayTotal, 0, len(dayMap))
	for _, t := range dayMap {
		result = append(result, *t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})
	return result
}

func aggregateTotalsByGroup(daily []provider.DailyStats) []groupTotal {
	if len(daily) == 0 {
		return nil
	}

	groupMap := make(map[string]*groupTotal)
	for _, d := range daily {
		name := strings.TrimSpace(d.Group)
		if name == "" {
			// Backward compatibility for inputs that do not set explicit group metadata.
			name = d.ProviderName
		}
		t, ok := groupMap[name]
		if !ok {
			t = &groupTotal{Name: name}
			groupMap[name] = t
		}
		t.Sessions += d.Sessions
		mergeTokenUsage(&t.TokenUsage, d.TokenUsage)
	}

	result := make([]groupTotal, 0, len(groupMap))
	for _, t := range groupMap {
		result = append(result, *t)
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].TokenUsage.Total()
		right := result[j].TokenUsage.Total()
		if left != right {
			return left > right
		}
		if result[i].Sessions != result[j].Sessions {
			return result[i].Sessions > result[j].Sessions
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func mergeTokenUsage(dst *provider.TokenUsage, src provider.TokenUsage) {
	dst.InputOther += src.InputOther
	dst.Output += src.Output
	dst.InputCacheRead += src.InputCacheRead
	dst.InputCacheCreate += src.InputCacheCreate
}

func shortDateLabel(date string) string {
	if len(date) == len("2006-01-02") {
		return date[5:]
	}
	return date
}

func trendBar(value, maxValue, width int) string {
	if width < 1 {
		width = 1
	}
	if maxValue <= 0 || value <= 0 {
		return strings.Repeat(".", width)
	}
	filled := int(math.Round(float64(value) / float64(maxValue) * float64(width)))
	if filled < 1 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
}

func groupColumnTitle(groupBy stats.AggregateDimension) string {
	if groupBy == stats.AggregateDimensionCLI {
		return "CLI"
	}
	return "Model"
}

func formatPercent(part, whole int) string {
	if whole <= 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", float64(part)*100/float64(whole))
}

func printDailyDashboard(daily []provider.DailyStats, unit tokenUnit, groupBy stats.AggregateDimension, topN int) {
	dateTotals := aggregateTotalsByDate(daily)
	groupTotals := aggregateTotalsByGroup(daily)

	printDailyTrend(dateTotals, unit)
	printGroupRanking(groupTotals, unit, groupBy)
	printTopGroupShare(groupTotals, unit, groupBy, topN)
}

func printDailyTrend(dateTotals []dayTotal, unit tokenUnit) {
	fmt.Fprintln(os.Stdout, "Daily Total Trend")
	if len(dateTotals) == 0 {
		fmt.Fprintln(os.Stdout, "No data for selected range.")
		fmt.Fprintln(os.Stdout)
		return
	}

	maxTotal := 0
	for _, d := range dateTotals {
		if total := d.TokenUsage.Total(); total > maxTotal {
			maxTotal = total
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(w, "Date\t")
	for _, d := range dateTotals {
		fmt.Fprintf(w, "%s\t", shortDateLabel(d.Date))
	}
	fmt.Fprintln(w)

	fmt.Fprint(w, "Total\t")
	for _, d := range dateTotals {
		fmt.Fprintf(w, "%s\t", formatTokenByUnit(d.TokenUsage.Total(), unit))
	}
	fmt.Fprintln(w)

	fmt.Fprint(w, "Bar\t")
	for _, d := range dateTotals {
		fmt.Fprintf(w, "%s\t", trendBar(d.TokenUsage.Total(), maxTotal, 10))
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Max\t%s\t\n", formatTokenByUnit(maxTotal, unit))
	w.Flush()
	fmt.Fprintln(os.Stdout)
}

func printGroupRanking(groupTotals []groupTotal, unit tokenUnit, groupBy stats.AggregateDimension) {
	groupTitle := groupColumnTitle(groupBy)
	fmt.Fprintf(os.Stdout, "%s Total Ranking\n", groupTitle)
	if len(groupTotals) == 0 {
		fmt.Fprintln(os.Stdout, "No groups for selected range.")
		fmt.Fprintln(os.Stdout)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Rank\t%s\tSessions\t%s\n", groupTitle, tokenHeader("Total", unit))
	for i, g := range groupTotals {
		fmt.Fprintf(
			w,
			"%d\t%s\t%d\t%s\n",
			i+1,
			g.Name,
			g.Sessions,
			formatTokenByUnit(g.TokenUsage.Total(), unit),
		)
	}
	w.Flush()
	fmt.Fprintln(os.Stdout)
}

func printTopGroupShare(groupTotals []groupTotal, unit tokenUnit, groupBy stats.AggregateDimension, topN int) {
	groupTitle := groupColumnTitle(groupBy)
	fmt.Fprintf(os.Stdout, "Top %d %s Share\n", topN, groupTitle)
	if len(groupTotals) == 0 {
		fmt.Fprintln(os.Stdout, "No groups for selected range.")
		return
	}

	if topN > len(groupTotals) {
		topN = len(groupTotals)
	}

	var totalUsage provider.TokenUsage
	for _, g := range groupTotals {
		mergeTokenUsage(&totalUsage, g.TokenUsage)
	}

	var topUsage provider.TokenUsage
	for i := 0; i < topN; i++ {
		mergeTokenUsage(&topUsage, groupTotals[i].TokenUsage)
	}

	fmt.Fprintf(
		os.Stdout,
		"Coverage: %s / %s (%s)\n",
		formatTokenByUnit(topUsage.Total(), unit),
		formatTokenByUnit(totalUsage.Total(), unit),
		formatPercent(topUsage.Total(), totalUsage.Total()),
	)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(
		w,
		"Rank\t%s\tShare\tSessions\t%s\t%s\t%s\t%s\t%s\n",
		groupTitle,
		tokenHeader("Total", unit),
		tokenHeader("Input", unit),
		tokenHeader("Output", unit),
		tokenHeader("Cache Read", unit),
		tokenHeader("Cache Create", unit),
	)
	for i := 0; i < topN; i++ {
		g := groupTotals[i]
		fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			i+1,
			g.Name,
			formatPercent(g.TokenUsage.Total(), totalUsage.Total()),
			g.Sessions,
			formatTokenByUnit(g.TokenUsage.Total(), unit),
			formatTokenByUnit(g.TokenUsage.InputOther, unit),
			formatTokenByUnit(g.TokenUsage.Output, unit),
			formatTokenByUnit(g.TokenUsage.InputCacheRead, unit),
			formatTokenByUnit(g.TokenUsage.InputCacheCreate, unit),
		)
	}

	w.Flush()
}
