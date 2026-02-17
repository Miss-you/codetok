package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Miss-you/codetok/provider"
	"github.com/Miss-you/codetok/provider/kimi"
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
	dailyCmd.Flags().String("base-dir", "", "Override default Kimi data directory")
	rootCmd.AddCommand(dailyCmd)
}

func runDaily(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	p := &kimi.Provider{}
	sessions, err := p.CollectSessions(baseDir)
	if err != nil {
		return fmt.Errorf("collecting sessions: %w", err)
	}

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

	sessions = stats.FilterByDateRange(sessions, since, until)
	daily := stats.AggregateByDay(sessions)

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
	fmt.Fprintln(w, "Date\tSessions\tInput\tOutput\tCache Read\tCache Create\tTotal")

	var totalSessions int
	var totalUsage provider.TokenUsage

	for _, d := range daily {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
			d.Date,
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

	fmt.Fprintf(w, "TOTAL\t%d\t%d\t%d\t%d\t%d\t%d\n",
		totalSessions,
		totalUsage.InputOther,
		totalUsage.Output,
		totalUsage.InputCacheRead,
		totalUsage.InputCacheCreate,
		totalUsage.Total(),
	)

	w.Flush()
}
