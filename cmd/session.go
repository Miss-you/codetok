package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
	_ "github.com/miss-you/codetok/provider/claude"
	_ "github.com/miss-you/codetok/provider/codex"
	_ "github.com/miss-you/codetok/provider/cursor"
	_ "github.com/miss-you/codetok/provider/kimi"
	"github.com/miss-you/codetok/stats"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Show per-session token usage",
	Long: `Show per-session token usage.

Session date filters match usage events in the selected date range, then group matching events by session. --timezone accepts an IANA timezone name; when omitted, codetok uses the local timezone.

Reporting commands read only local session files and Cursor CSV exports already on disk. They never trigger implicit Cursor login or sync.

Codex reads $CODEX_HOME/sessions when CODEX_HOME is set, otherwise ~/.codex/sessions.

By default Cursor reporting scans legacy CSV files in ~/.codetok/cursor/ plus imports/ and synced/ subdirectories. Use --cursor-dir to scan only a custom local directory.`,
	RunE: runSession,
}

func init() {
	sessionCmd.Flags().Bool("json", false, "Output as JSON")
	sessionCmd.Flags().String("since", "", "Start date filter (format: 2006-01-02)")
	sessionCmd.Flags().String("until", "", "End date filter (format: 2006-01-02)")
	sessionCmd.Flags().String("timezone", "", "Timezone for date filters (IANA name, default: local)")
	sessionCmd.Flags().String("provider", "", "Filter by provider name (e.g. kimi, claude, codex, cursor)")
	sessionCmd.Flags().String("base-dir", "", "Override default data directory (applies to all providers)")
	sessionCmd.Flags().String("kimi-dir", "", "Override Kimi data directory")
	sessionCmd.Flags().String("claude-dir", "", "Override Claude Code data directory")
	sessionCmd.Flags().String("codex-dir", "", "Override Codex CLI data directory")
	sessionCmd.Flags().String("cursor-dir", "", "Override Cursor CSV directory; scans only this local path and skips default Cursor imports/synced roots")
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
	return runSessionWithProviders(cmd, args, provider.Registry())
}

func runSessionWithProviders(cmd *cobra.Command, args []string, providers []provider.Provider) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	sinceStr, _ := cmd.Flags().GetString("since")
	untilStr, _ := cmd.Flags().GetString("until")
	timezoneStr, _ := cmd.Flags().GetString("timezone")

	loc, err := resolveTimezone(timezoneStr)
	if err != nil {
		return err
	}

	sinceDate, untilDate, since, until, err := resolveSessionEventFilterRange(sinceStr, untilStr, loc)
	if err != nil {
		return err
	}

	events, err := collectUsageEventsFromProvidersInRange(cmd, providers, provider.UsageEventCollectOptions{
		Since:    since,
		Until:    until,
		Location: loc,
	})
	if err != nil {
		return err
	}
	events = stats.FilterEventsByDateRange(events, sinceDate, untilDate, loc)
	allSessions := aggregateSessionEvents(events)

	if jsonOutput {
		out := make([]sessionJSON, len(allSessions))
		for i, s := range allSessions {
			out[i] = sessionJSON{
				SessionID:    s.SessionID,
				ProviderName: s.ProviderName,
				Title:        s.Title,
				Date:         sessionOutputDate(s.StartTime, loc),
				Turns:        s.Turns,
				TokenUsage:   s.TokenUsage,
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	printSessionTableWithLocation(allSessions, loc)
	return nil
}

func resolveSessionEventFilterDates(sinceStr, untilStr string, loc *time.Location) (string, string, error) {
	sinceDate, untilDate, _, _, err := resolveSessionEventFilterRange(sinceStr, untilStr, loc)
	return sinceDate, untilDate, err
}

func resolveSessionEventFilterRange(sinceStr, untilStr string, loc *time.Location) (string, string, time.Time, time.Time, error) {
	if loc == nil {
		loc = time.Local
	}

	var sinceDate, untilDate string
	var since, until time.Time
	if strings.TrimSpace(sinceStr) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(sinceStr), loc)
		if err != nil {
			return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid --since date: %w", err)
		}
		since = parsed
		sinceDate = parsed.Format("2006-01-02")
	}
	if strings.TrimSpace(untilStr) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(untilStr), loc)
		if err != nil {
			return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid --until date: %w", err)
		}
		untilDate = parsed.Format("2006-01-02")
		until = time.Date(parsed.Year(), parsed.Month(), parsed.Day()+1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)
	}
	return sinceDate, untilDate, since, until, nil
}

func aggregateSessionEvents(events []provider.UsageEvent) []provider.SessionInfo {
	if len(events) == 0 {
		return nil
	}

	ordered := append([]provider.UsageEvent(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if !ordered[i].Timestamp.Equal(ordered[j].Timestamp) {
			if ordered[i].Timestamp.IsZero() {
				return false
			}
			if ordered[j].Timestamp.IsZero() {
				return true
			}
			return ordered[i].Timestamp.Before(ordered[j].Timestamp)
		}
		if ordered[i].ProviderName != ordered[j].ProviderName {
			return ordered[i].ProviderName < ordered[j].ProviderName
		}
		return sessionEventDisplayID(ordered[i]) < sessionEventDisplayID(ordered[j])
	})

	sessionMap := make(map[string]*provider.SessionInfo)
	for _, event := range ordered {
		key := sessionEventGroupKey(event)
		session, ok := sessionMap[key]
		if !ok {
			session = &provider.SessionInfo{
				ProviderName: strings.TrimSpace(event.ProviderName),
				SessionID:    sessionEventDisplayID(event),
				StartTime:    event.Timestamp,
				EndTime:      event.Timestamp,
			}
			sessionMap[key] = session
		}

		if session.ProviderName == "" {
			session.ProviderName = strings.TrimSpace(event.ProviderName)
		}
		if session.SessionID == "" || session.SessionID == "unknown" {
			session.SessionID = sessionEventDisplayID(event)
		}
		if session.ModelName == "" {
			session.ModelName = strings.TrimSpace(event.ModelName)
		}
		if session.Title == "" {
			session.Title = event.Title
		}
		if session.WorkDirHash == "" {
			session.WorkDirHash = strings.TrimSpace(event.WorkDirHash)
		}
		if session.StartTime.IsZero() || (!event.Timestamp.IsZero() && event.Timestamp.Before(session.StartTime)) {
			session.StartTime = event.Timestamp
		}
		if event.Timestamp.After(session.EndTime) {
			session.EndTime = event.Timestamp
		}
		session.Turns++
		mergeTokenUsage(&session.TokenUsage, event.TokenUsage)
	}

	rows := make([]provider.SessionInfo, 0, len(sessionMap))
	for _, session := range sessionMap {
		rows = append(rows, *session)
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].StartTime.Equal(rows[j].StartTime) {
			if rows[i].StartTime.IsZero() {
				return false
			}
			if rows[j].StartTime.IsZero() {
				return true
			}
			return rows[i].StartTime.Before(rows[j].StartTime)
		}
		if rows[i].ProviderName != rows[j].ProviderName {
			return rows[i].ProviderName < rows[j].ProviderName
		}
		return rows[i].SessionID < rows[j].SessionID
	})
	return rows
}

func sessionEventGroupKey(event provider.UsageEvent) string {
	providerName := strings.TrimSpace(event.ProviderName)
	if sessionID := strings.TrimSpace(event.SessionID); sessionID != "" {
		return providerName + "\x00session\x00" + sessionID
	}
	if sourcePath := strings.TrimSpace(event.SourcePath); sourcePath != "" {
		return providerName + "\x00source\x00" + sourcePath
	}
	if eventID := strings.TrimSpace(event.EventID); eventID != "" {
		return providerName + "\x00event\x00" + eventID
	}
	return providerName + "\x00anonymous"
}

func sessionEventDisplayID(event provider.UsageEvent) string {
	if sessionID := strings.TrimSpace(event.SessionID); sessionID != "" {
		return sessionID
	}
	if sourcePath := strings.TrimSpace(event.SourcePath); sourcePath != "" {
		return sourcePath
	}
	if eventID := strings.TrimSpace(event.EventID); eventID != "" {
		return eventID
	}
	return "unknown"
}

func sessionOutputDate(ts time.Time, loc *time.Location) string {
	if ts.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.Local
	}
	return ts.In(loc).Format("2006-01-02")
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func printSessionTableWithLocation(sessions []provider.SessionInfo, loc *time.Location) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Date\tProvider\tSession\tTitle\tInput\tOutput\tTotal")

	var totalUsage provider.TokenUsage

	for _, s := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%d\n",
			sessionOutputDate(s.StartTime, loc),
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
