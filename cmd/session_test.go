package cmd

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
)

func TestRunSession_JSONFiltersByUsageEventDate(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{
			name: "codex",
			sessions: []provider.SessionInfo{{
				ProviderName: "codex",
				ModelName:    "legacy-model",
				SessionID:    "cross-day-session",
				Title:        "legacy session",
				StartTime:    time.Date(2026, 4, 15, 23, 50, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 999},
			}},
		},
		events: []provider.UsageEvent{
			{
				ProviderName: "codex",
				ModelName:    "gpt-5.4",
				SessionID:    "cross-day-session",
				Title:        "Cross-day work",
				Timestamp:    time.Date(2026, 4, 15, 23, 50, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 100},
			},
			{
				ProviderName: "codex",
				ModelName:    "gpt-5.4",
				SessionID:    "cross-day-session",
				Title:        "Cross-day work",
				Timestamp:    time.Date(2026, 4, 16, 0, 10, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 200, Output: 20},
			},
			{
				ProviderName: "codex",
				ModelName:    "gpt-5.4",
				SessionID:    "cross-day-session",
				Title:        "Cross-day work",
				Timestamp:    time.Date(2026, 4, 16, 22, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputCacheRead: 50, Output: 5},
			},
			{
				ProviderName: "codex",
				ModelName:    "gpt-5.4",
				SessionID:    "cross-day-session",
				Title:        "Cross-day work",
				Timestamp:    time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 400},
			},
		},
	}
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("since", "2026-04-16"); err != nil {
		t.Fatalf("setting --since: %v", err)
	}
	if err := cmd.Flags().Set("until", "2026-04-16"); err != nil {
		t.Fatalf("setting --until: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "UTC"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runSessionWithProviders(cmd, nil, []provider.Provider{eventProvider}); err != nil {
			t.Fatalf("runSessionWithProviders returned error: %v", err)
		}
	})

	got := decodeSessionJSON(t, output)
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want one in-range session: %#v", len(got), got)
	}
	if got[0].SessionID != "cross-day-session" || got[0].ProviderName != "codex" {
		t.Fatalf("session identity mismatch: %#v", got[0])
	}
	if got[0].Date != "2026-04-16" {
		t.Fatalf("date = %q, want first included event date 2026-04-16", got[0].Date)
	}
	if got[0].Title != "Cross-day work" {
		t.Fatalf("metadata mismatch: %#v", got[0])
	}
	if got[0].TokenUsage.Total() != 275 {
		t.Fatalf("total = %d, want only filtered event total 275", got[0].TokenUsage.Total())
	}
}

func TestRunSession_ExplicitDateRangeUsesRangeAwareCollector(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		eventErr:            errors.New("full-history collector called"),
		rangeEvents: []provider.UsageEvent{
			{
				ProviderName: "codex",
				SessionID:    "cross-day-session",
				Title:        "Cross-day work",
				Timestamp:    time.Date(2026, 4, 15, 23, 50, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 100},
			},
			{
				ProviderName: "codex",
				SessionID:    "cross-day-session",
				Title:        "Cross-day work",
				Timestamp:    time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 200, Output: 20},
			},
		},
	}
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("since", "2026-04-16"); err != nil {
		t.Fatalf("setting --since: %v", err)
	}
	if err := cmd.Flags().Set("until", "2026-04-16"); err != nil {
		t.Fatalf("setting --until: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "UTC"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runSessionWithProviders(cmd, nil, []provider.Provider{eventProvider}); err != nil {
			t.Fatalf("runSessionWithProviders returned error: %v", err)
		}
	})

	got := decodeSessionJSON(t, output)
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want one in-range session: %#v", len(got), got)
	}
	if got[0].TokenUsage.Total() != 220 {
		t.Fatalf("total = %d, want only in-window event total 220", got[0].TokenUsage.Total())
	}
	if len(eventProvider.seenEventDirs) != 0 {
		t.Fatalf("full-history collector should not be called, seen %v", eventProvider.seenEventDirs)
	}
	if len(eventProvider.seenRangeOpts) != 1 {
		t.Fatalf("range opts seen %d times, want 1", len(eventProvider.seenRangeOpts))
	}
	wantSince := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	wantUntil := time.Date(2026, 4, 16, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
	gotOpts := eventProvider.seenRangeOpts[0]
	if !gotOpts.Since.Equal(wantSince) || !gotOpts.Until.Equal(wantUntil) || gotOpts.Location != time.UTC {
		t.Fatalf("range opts = %+v, want since=%v until=%v UTC", gotOpts, wantSince, wantUntil)
	}
}

func TestRunSession_UntilOnlyPassesFullLocalDayToRangeAwareCollector(t *testing.T) {
	loc := mustLoadLocation(t, "Asia/Shanghai")
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "cursor"},
		eventErr:            errors.New("full-history collector called"),
		rangeEvents: []provider.UsageEvent{{
			ProviderName: "cursor",
			SessionID:    "until-boundary",
			Timestamp:    time.Date(2026, 4, 16, 23, 30, 0, 0, loc),
			TokenUsage:   provider.TokenUsage{InputOther: 10},
		}},
	}
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("until", "2026-04-16"); err != nil {
		t.Fatalf("setting --until: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "Asia/Shanghai"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runSessionWithProviders(cmd, nil, []provider.Provider{eventProvider}); err != nil {
			t.Fatalf("runSessionWithProviders returned error: %v", err)
		}
	})
	got := decodeSessionJSON(t, output)
	if len(got) != 1 || got[0].SessionID != "until-boundary" {
		t.Fatalf("sessions = %#v, want boundary event included", got)
	}
	gotOpts := eventProvider.seenRangeOpts[0]
	wantUntil := time.Date(2026, 4, 16, 23, 59, 59, int(time.Second-time.Nanosecond), loc)
	if !gotOpts.Since.IsZero() || !gotOpts.Until.Equal(wantUntil) || gotOpts.Location.String() != "Asia/Shanghai" {
		t.Fatalf("range opts = %+v, want zero since until=%v Asia/Shanghai", gotOpts, wantUntil)
	}
}

func TestRunSession_NoDateRangeUsesFullHistoryCollector(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{{
			ProviderName: "codex",
			SessionID:    "full-history",
			Timestamp:    time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 10},
		}},
		rangeErr: errors.New("range collector should not be called"),
	}
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runSessionWithProviders(cmd, nil, []provider.Provider{eventProvider}); err != nil {
			t.Fatalf("runSessionWithProviders returned error: %v", err)
		}
	})
	got := decodeSessionJSON(t, output)
	if len(got) != 1 || got[0].SessionID != "full-history" {
		t.Fatalf("sessions = %#v, want full-history session", got)
	}
	if len(eventProvider.seenRangeDirs) != 0 {
		t.Fatalf("range collector should not be called without date filters, seen %v", eventProvider.seenRangeDirs)
	}
}

func TestRunSession_JSONTimezoneFiltersByLocalEventDate(t *testing.T) {
	event := provider.UsageEvent{
		ProviderName: "codex",
		ModelName:    "gpt-5.4",
		SessionID:    "timezone-session",
		Timestamp:    time.Date(2026, 4, 15, 18, 0, 0, 0, time.UTC),
		TokenUsage:   provider.TokenUsage{InputOther: 123},
	}

	run := func(timezone string) []sessionJSON {
		cmd := newSessionTestCommand()
		if err := cmd.Flags().Set("json", "true"); err != nil {
			t.Fatalf("setting --json: %v", err)
		}
		if err := cmd.Flags().Set("since", "2026-04-16"); err != nil {
			t.Fatalf("setting --since: %v", err)
		}
		if err := cmd.Flags().Set("until", "2026-04-16"); err != nil {
			t.Fatalf("setting --until: %v", err)
		}
		if err := cmd.Flags().Set("timezone", timezone); err != nil {
			t.Fatalf("setting --timezone: %v", err)
		}
		eventProvider := &collectTestUsageEventProvider{
			collectTestProvider: collectTestProvider{name: "codex"},
			events:              []provider.UsageEvent{event},
		}

		output := captureStdout(t, func() {
			if err := runSessionWithProviders(cmd, nil, []provider.Provider{eventProvider}); err != nil {
				t.Fatalf("runSessionWithProviders returned error: %v", err)
			}
		})
		return decodeSessionJSON(t, output)
	}

	gotUTC := run("UTC")
	if len(gotUTC) != 0 {
		t.Fatalf("UTC rows = %#v, want event excluded from 2026-04-16 UTC", gotUTC)
	}

	gotShanghai := run("Asia/Shanghai")
	if len(gotShanghai) != 1 {
		t.Fatalf("got %d sessions, want Shanghai-local event included: %#v", len(gotShanghai), gotShanghai)
	}
	if gotShanghai[0].Date != "2026-04-16" || gotShanghai[0].TokenUsage.Total() != 123 {
		t.Fatalf("session = %#v, want 2026-04-16 total 123", gotShanghai[0])
	}
}

func TestRunSession_TableUsesEventAggregation(t *testing.T) {
	eventProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{
			{
				ProviderName: "codex",
				SessionID:    "table-session",
				Title:        "Table work",
				Timestamp:    time.Date(2026, 4, 15, 23, 0, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 999},
			},
			{
				ProviderName: "codex",
				SessionID:    "table-session",
				Title:        "Table work",
				Timestamp:    time.Date(2026, 4, 16, 0, 30, 0, 0, time.UTC),
				TokenUsage:   provider.TokenUsage{InputOther: 200, Output: 20},
			},
		},
	}
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("since", "2026-04-16"); err != nil {
		t.Fatalf("setting --since: %v", err)
	}
	if err := cmd.Flags().Set("until", "2026-04-16"); err != nil {
		t.Fatalf("setting --until: %v", err)
	}
	if err := cmd.Flags().Set("timezone", "UTC"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runSessionWithProviders(cmd, nil, []provider.Provider{eventProvider}); err != nil {
			t.Fatalf("runSessionWithProviders returned error: %v", err)
		}
	})

	assertContainsAll(t, output, "Date", "Provider", "Session", "2026-04-16", "table-session", "Table work", "220")
	if strings.Contains(output, "999") || strings.Contains(output, "1219") {
		t.Fatalf("table output included out-of-range usage:\n%s", output)
	}
}

func TestRunSession_InvalidTimezone(t *testing.T) {
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("timezone", "not/a-zone"); err != nil {
		t.Fatalf("setting --timezone: %v", err)
	}
	if err := cmd.Flags().Set("provider", "nonexistent"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	err := runSessionWithProviders(cmd, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid --timezone") {
		t.Fatalf("expected invalid --timezone error, got: %v", err)
	}
}

func TestRunSession_UsesEventCollectorProviderOverrides(t *testing.T) {
	codexProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "codex"},
		events: []provider.UsageEvent{{
			ProviderName: "codex",
			SessionID:    "from-codex",
			Timestamp:    time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 1},
		}},
	}
	claudeProvider := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "claude"},
		events: []provider.UsageEvent{{
			ProviderName: "claude",
			SessionID:    "from-claude",
			Timestamp:    time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 1},
		}},
	}
	cmd := newSessionTestCommand()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	if err := cmd.Flags().Set("provider", "codex"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}
	if err := cmd.Flags().Set("base-dir", "/shared"); err != nil {
		t.Fatalf("setting --base-dir: %v", err)
	}
	if err := cmd.Flags().Set("codex-dir", "/codex-only"); err != nil {
		t.Fatalf("setting --codex-dir: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runSessionWithProviders(cmd, nil, []provider.Provider{codexProvider, claudeProvider}); err != nil {
			t.Fatalf("runSessionWithProviders returned error: %v", err)
		}
	})

	got := decodeSessionJSON(t, output)
	if len(got) != 1 || got[0].SessionID != "from-codex" {
		t.Fatalf("sessions = %#v, want only codex event", got)
	}
	if len(codexProvider.seenEventDirs) != 1 || codexProvider.seenEventDirs[0] != "/codex-only" {
		t.Fatalf("codex event dirs = %v, want [/codex-only]", codexProvider.seenEventDirs)
	}
	if len(claudeProvider.seenEventDirs) != 0 {
		t.Fatalf("claude should not be collected when filtered, seen %v", claudeProvider.seenEventDirs)
	}
}

func TestResolveSessionEventFilterDatesInvalid(t *testing.T) {
	_, _, err := resolveSessionEventFilterDates("bad", "", time.UTC)
	if err == nil || !strings.Contains(err.Error(), "invalid --since date") {
		t.Fatalf("expected invalid --since date error, got: %v", err)
	}

	_, _, err = resolveSessionEventFilterDates("", "bad", time.UTC)
	if err == nil || !strings.Contains(err.Error(), "invalid --until date") {
		t.Fatalf("expected invalid --until date error, got: %v", err)
	}
}

func TestAggregateSessionEventsTracksFirstAndLastIncludedEvents(t *testing.T) {
	events := []provider.UsageEvent{
		{
			ProviderName: "codex",
			ModelName:    "gpt-5.4",
			SessionID:    "same-id",
			Title:        "later title",
			Timestamp:    time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{Output: 20},
		},
		{
			ProviderName: "codex",
			ModelName:    "gpt-5.4",
			SessionID:    "same-id",
			Title:        "earlier title",
			Timestamp:    time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 100},
		},
		{
			ProviderName: "claude",
			ModelName:    "gpt-5.4",
			SessionID:    "same-id",
			Title:        "provider boundary",
			Timestamp:    time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 500},
		},
	}

	got := aggregateSessionEvents(events)
	if len(got) != 2 {
		t.Fatalf("got %d sessions, want provider/session boundary preserved: %#v", len(got), got)
	}
	first := got[0]
	if first.ProviderName != "codex" || first.SessionID != "same-id" {
		t.Fatalf("first session identity mismatch: %#v", first)
	}
	if !first.StartTime.Equal(time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("StartTime = %v, want first included event timestamp", first.StartTime)
	}
	if !first.EndTime.Equal(time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("EndTime = %v, want last included event timestamp", first.EndTime)
	}
	if first.Title != "earlier title" || first.ModelName != "gpt-5.4" {
		t.Fatalf("metadata mismatch: %#v", first)
	}
	if first.TokenUsage.Total() != 120 {
		t.Fatalf("total = %d, want 120", first.TokenUsage.Total())
	}
}

func TestAggregateSessionEventsUsesSourcePathAndEventIDFallbacks(t *testing.T) {
	events := []provider.UsageEvent{
		{
			ProviderName: "codex",
			SourcePath:   "/logs/a.jsonl",
			Timestamp:    time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 10},
		},
		{
			ProviderName: "codex",
			SourcePath:   "/logs/b.jsonl",
			Timestamp:    time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 20},
		},
		{
			ProviderName: "codex",
			EventID:      "event-only",
			Timestamp:    time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC),
			TokenUsage:   provider.TokenUsage{InputOther: 30},
		},
	}

	got := aggregateSessionEvents(events)
	if len(got) != 3 {
		t.Fatalf("got %d sessions, want fallback keys to stay separate: %#v", len(got), got)
	}
	if got[0].SessionID != "/logs/a.jsonl" || got[0].TokenUsage.Total() != 10 {
		t.Fatalf("first fallback row mismatch: %#v", got[0])
	}
	if got[1].SessionID != "/logs/b.jsonl" || got[1].TokenUsage.Total() != 20 {
		t.Fatalf("second fallback row mismatch: %#v", got[1])
	}
	if got[2].SessionID != "event-only" || got[2].TokenUsage.Total() != 30 {
		t.Fatalf("event fallback row mismatch: %#v", got[2])
	}
}

func decodeSessionJSON(t *testing.T, output string) []sessionJSON {
	t.Helper()
	var got []sessionJSON
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decoding session json: %v\noutput: %s", err, output)
	}
	return got
}

func newSessionTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("until", "", "")
	cmd.Flags().String("timezone", "", "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("base-dir", "", "")
	cmd.Flags().String("kimi-dir", "", "")
	cmd.Flags().String("claude-dir", "", "")
	cmd.Flags().String("codex-dir", "", "")
	cmd.Flags().String("cursor-dir", "", "")
	return cmd
}
