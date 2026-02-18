package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
	_ "github.com/miss-you/codetok/provider/claude"
	_ "github.com/miss-you/codetok/provider/codex"
	_ "github.com/miss-you/codetok/provider/kimi"
	"github.com/miss-you/codetok/stats"
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
	sessionCmd.Flags().String("provider", "", "Filter by provider name (e.g. kimi, claude, codex)")
	sessionCmd.Flags().String("base-dir", "", "Override default data directory (applies to all providers)")
	sessionCmd.Flags().String("kimi-dir", "", "Override Kimi data directory")
	sessionCmd.Flags().String("claude-dir", "", "Override Claude Code data directory")
	sessionCmd.Flags().String("codex-dir", "", "Override Codex CLI data directory")
	rootCmd.AddCommand(sessionCmd)
}

// sessionJSON is the JSON output representation of a session.
type sessionJSON struct {
	SessionID    string              `json:"session_id"`
	ProviderName string              `json:"provider"`
	Title        string              `json:"title"`
	Date         string              `json:"date"`
	Turns        int                 `json:"turns"`
	TokenUsage   provider.TokenUsage `json:"token_usage"`
}

func runSession(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	providerFilter, _ := cmd.Flags().GetString("provider")
	baseDir, _ := cmd.Flags().GetString("base-dir")

	providers := provider.FilterProviders(provider.Registry(), providerFilter)

	var allSessions []provider.SessionInfo
	for _, p := range providers {
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
		until = until.Add(24*time.Hour - time.Nanosecond)
	}

	allSessions = stats.FilterByDateRange(allSessions, since, until)

	if jsonOutput {
		out := make([]sessionJSON, len(allSessions))
		for i, s := range allSessions {
			date := ""
			if !s.StartTime.IsZero() {
				date = s.StartTime.Format("2006-01-02")
			}
			out[i] = sessionJSON{
				SessionID:    s.SessionID,
				ProviderName: s.ProviderName,
				Title:        s.Title,
				Date:         date,
				Turns:        s.Turns,
				TokenUsage:   s.TokenUsage,
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	printSessionTable(allSessions)
	return nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func printSessionTable(sessions []provider.SessionInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Date\tProvider\tSession\tTitle\tInput\tOutput\tTotal")

	var totalUsage provider.TokenUsage

	for _, s := range sessions {
		date := ""
		if !s.StartTime.IsZero() {
			date = s.StartTime.Format("2006-01-02")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%d\n",
			date,
			s.ProviderName,
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

	fmt.Fprintf(w, "TOTAL\t\t\t\t%d\t%d\t%d\n",
		totalUsage.TotalInput(),
		totalUsage.Output,
		totalUsage.Total(),
	)

	w.Flush()
}
