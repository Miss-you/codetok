package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Miss-you/codetok/provider"
	_ "github.com/Miss-you/codetok/provider/claude"
	_ "github.com/Miss-you/codetok/provider/codex"
	_ "github.com/Miss-you/codetok/provider/kimi"
	"github.com/Miss-you/codetok/stats"
	"github.com/spf13/cobra"
)

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Show daily token usage breakdown",
	RunE:  runDaily,
}

func init() {
	dailyCmd.Flags().Bool("json", false, "Output as JSON")
	dailyCmd.Flags().String("since", "", "Start date filter (format: 2006-01-02)")
	dailyCmd.Flags().String("until", "", "End date filter (format: 2006-01-02)")
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

	var err error
	var since, until time.Time
	if sinceStr != "" {
		since, err = time.Parse("2006-01-02", sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since date: %w", err)
		}
	}
	if untilStr != "" {
		until, err = time.Parse("2006-01-02", untilStr)
		if err != nil {
			return fmt.Errorf("invalid --until date: %w", err)
		}
		// Include the entire "until" day
		until = until.Add(24*time.Hour - time.Nanosecond)
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

	printDailyTable(daily)
	return nil
}

func printDailyTable(daily []provider.DailyStats) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Date\tProvider\tSessions\tInput\tOutput\tCache Read\tCache Create\tTotal")

	var totalSessions int
	var totalUsage provider.TokenUsage

	for _, d := range daily {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			d.Date,
			d.ProviderName,
			d.Sessions,
			d.TokenUsage.InputOther,
			d.TokenUsage.Output,
			d.TokenUsage.InputCacheRead,
			d.TokenUsage.InputCacheCreate,
			d.TokenUsage.Total(),
		)
		totalSessions += d.Sessions
		totalUsage.InputOther += d.TokenUsage.InputOther
		totalUsage.Output += d.TokenUsage.Output
		totalUsage.InputCacheRead += d.TokenUsage.InputCacheRead
		totalUsage.InputCacheCreate += d.TokenUsage.InputCacheCreate
	}

	fmt.Fprintf(w, "TOTAL\t\t%d\t%d\t%d\t%d\t%d\t%d\n",
		totalSessions,
		totalUsage.InputOther,
		totalUsage.Output,
		totalUsage.InputCacheRead,
		totalUsage.InputCacheCreate,
		totalUsage.Total(),
	)

	w.Flush()
}
