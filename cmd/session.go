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

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Show per-session token usage",
	RunE:  runSession,
}

func init() {
	sessionCmd.Flags().Bool("json", false, "Output as JSON")
	sessionCmd.Flags().String("since", "", "Start date filter (format: 2006-01-02)")
	sessionCmd.Flags().String("until", "", "End date filter (format: 2006-01-02)")
	sessionCmd.Flags().String("base-dir", "", "Override default Kimi data directory")
	rootCmd.AddCommand(sessionCmd)
}

// sessionJSON is the JSON output representation of a session.
type sessionJSON struct {
	SessionID  string              `json:"session_id"`
	Title      string              `json:"title"`
	Date       string              `json:"date"`
	Turns      int                 `json:"turns"`
	TokenUsage provider.TokenUsage `json:"token_usage"`
}

func runSession(cmd *cobra.Command, args []string) error {
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
		until = until.Add(24*time.Hour - time.Nanosecond)
	}

	sessions = stats.FilterByDateRange(sessions, since, until)

	if jsonOutput {
		out := make([]sessionJSON, len(sessions))
		for i, s := range sessions {
			out[i] = sessionJSON{
				SessionID:  s.SessionID,
				Title:      s.Title,
				Date:       s.StartTime.Format("2006-01-02"),
				Turns:      s.Turns,
				TokenUsage: s.TokenUsage,
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	printSessionTable(sessions)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func printSessionTable(sessions []provider.SessionInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Date\tSession\tTitle\tInput\tOutput\tTotal")

	var totalUsage provider.TokenUsage

	for _, s := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\n",
			s.StartTime.Format("2006-01-02"),
			s.SessionID,
			truncate(s.Title, 40),
			s.TokenUsage.TotalInput(),
			s.TokenUsage.Output,
			s.TokenUsage.Total(),
		)
		totalUsage.InputOther += s.TokenUsage.InputOther
		totalUsage.Output += s.TokenUsage.Output
		totalUsage.InputCacheRead += s.TokenUsage.InputCacheRead
		totalUsage.InputCacheCreate += s.TokenUsage.InputCacheCreate
	}

	fmt.Fprintf(w, "TOTAL\t\t\t%d\t%d\t%d\n",
		totalUsage.TotalInput(),
		totalUsage.Output,
		totalUsage.Total(),
	)

	w.Flush()
}
