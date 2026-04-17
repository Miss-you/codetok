package cursor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

var _ provider.UsageEventProvider = (*Provider)(nil)

func TestParseUsageCSV_ValidExport(t *testing.T) {
	sessions, err := parseUsageCSV(filepath.Join("testdata", "usage-export.csv"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	first := sessions[0]
	if first.ProviderName != "cursor" {
		t.Fatalf("ProviderName = %q, want %q", first.ProviderName, "cursor")
	}
	if first.SessionID != "usage-export:1" {
		t.Fatalf("SessionID = %q, want %q", first.SessionID, "usage-export:1")
	}
	if first.ModelName != "auto" {
		t.Fatalf("ModelName = %q, want %q", first.ModelName, "auto")
	}
	if first.Title != "Included auto" {
		t.Fatalf("Title = %q, want %q", first.Title, "Included auto")
	}
	if first.Turns != 1 {
		t.Fatalf("Turns = %d, want 1", first.Turns)
	}

	wantStart := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	if !first.StartTime.Equal(wantStart) {
		t.Fatalf("StartTime = %v, want %v", first.StartTime, wantStart)
	}
	if !first.EndTime.Equal(wantStart) {
		t.Fatalf("EndTime = %v, want %v", first.EndTime, wantStart)
	}

	if first.TokenUsage.InputOther != 775 {
		t.Fatalf("InputOther = %d, want 775", first.TokenUsage.InputOther)
	}
	if first.TokenUsage.InputCacheCreate != 28342 {
		t.Fatalf("InputCacheCreate = %d, want 28342", first.TokenUsage.InputCacheCreate)
	}
	if first.TokenUsage.InputCacheRead != 105891 {
		t.Fatalf("InputCacheRead = %d, want 105891", first.TokenUsage.InputCacheRead)
	}
	if first.TokenUsage.Output != 21282 {
		t.Fatalf("Output = %d, want 21282", first.TokenUsage.Output)
	}
	if first.TokenUsage.Total() != 156290 {
		t.Fatalf("Total = %d, want 156290", first.TokenUsage.Total())
	}

	second := sessions[1]
	if second.SessionID != "usage-export:2" {
		t.Fatalf("SessionID = %q, want %q", second.SessionID, "usage-export:2")
	}
	if second.ModelName != "gpt-5-codex" {
		t.Fatalf("ModelName = %q, want %q", second.ModelName, "gpt-5-codex")
	}
	if second.TokenUsage.InputOther != 8263 {
		t.Fatalf("InputOther = %d, want 8263", second.TokenUsage.InputOther)
	}
	if second.TokenUsage.InputCacheCreate != 0 {
		t.Fatalf("InputCacheCreate = %d, want 0", second.TokenUsage.InputCacheCreate)
	}
	if second.TokenUsage.InputCacheRead != 66964 {
		t.Fatalf("InputCacheRead = %d, want 66964", second.TokenUsage.InputCacheRead)
	}
	if second.TokenUsage.Output != 1612 {
		t.Fatalf("Output = %d, want 1612", second.TokenUsage.Output)
	}
	if second.TokenUsage.Total() != 76839 {
		t.Fatalf("Total = %d, want 76839", second.TokenUsage.Total())
	}
}

func TestCollectUsageEvents_ValidExportMapsRowsToEvents(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "usage.csv")
	content := `Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2026-02-17T10:00:00Z","Included","auto","No","28342","775","105891","21282","999999","0.19"
"2026-02-18T11:30:00+08:00","On-Demand","gpt-5-codex","No","0","8263","66964","1612","76839","0.03"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	events, err := p.CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	first := events[0]
	if first.ProviderName != "cursor" {
		t.Fatalf("ProviderName = %q, want cursor", first.ProviderName)
	}
	if first.SessionID != "usage:1" {
		t.Fatalf("SessionID = %q, want usage:1", first.SessionID)
	}
	if first.EventID != "usage:1" {
		t.Fatalf("EventID = %q, want usage:1", first.EventID)
	}
	if first.SourcePath != path {
		t.Fatalf("SourcePath = %q, want %q", first.SourcePath, path)
	}
	if first.ModelName != "auto" {
		t.Fatalf("ModelName = %q, want auto", first.ModelName)
	}
	if first.Title != "Included auto" {
		t.Fatalf("Title = %q, want Included auto", first.Title)
	}
	wantFirstTimestamp := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	if !first.Timestamp.Equal(wantFirstTimestamp) {
		t.Fatalf("Timestamp = %v, want %v", first.Timestamp, wantFirstTimestamp)
	}
	if first.TokenUsage.InputCacheCreate != 28342 {
		t.Fatalf("InputCacheCreate = %d, want 28342", first.TokenUsage.InputCacheCreate)
	}
	if first.TokenUsage.InputOther != 775 {
		t.Fatalf("InputOther = %d, want 775", first.TokenUsage.InputOther)
	}
	if first.TokenUsage.InputCacheRead != 105891 {
		t.Fatalf("InputCacheRead = %d, want 105891", first.TokenUsage.InputCacheRead)
	}
	if first.TokenUsage.Output != 21282 {
		t.Fatalf("Output = %d, want 21282", first.TokenUsage.Output)
	}

	second := events[1]
	if second.SessionID != "usage:2" {
		t.Fatalf("second SessionID = %q, want usage:2", second.SessionID)
	}
	if second.EventID != "usage:2" {
		t.Fatalf("second EventID = %q, want usage:2", second.EventID)
	}
	if got, want := second.Timestamp.Format(time.RFC3339), "2026-02-18T11:30:00+08:00"; got != want {
		t.Fatalf("second Timestamp = %q, want %q", got, want)
	}
	if second.TokenUsage.Total() != 76839 {
		t.Fatalf("second Total = %d, want 76839", second.TokenUsage.Total())
	}
}

func TestCollectUsageEvents_SkipsInvalidCSVFileAndKeepsDeterministicOrder(t *testing.T) {
	baseDir := t.TempDir()

	invalid := `Date,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read
"2026-02-17T10:00:00Z","broken","1","2","3"
`
	first := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens
"not-a-date","Included","broken","1","2","3","4"
"2026-02-21T09:15:00Z","Included","model-b","1","2","3","4"
`
	second := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens
"2026-02-20T09:15:00Z","Included","model-a","5","6","7","8"
"2026-02-21T09:15:00Z","Included","model-c","9","10","11","12"
`

	if err := os.WriteFile(filepath.Join(baseDir, "zzz-invalid.csv"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "bbb.csv"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "aaa.csv"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	events, err := p.CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	gotIDs := []string{events[0].SessionID, events[1].SessionID, events[2].SessionID}
	wantIDs := []string{"aaa:1", "aaa:2", "bbb:2"}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("sessionIDs[%d] = %q, want %q", i, gotIDs[i], wantIDs[i])
		}
	}
	for _, event := range events {
		if event.SourcePath == "" {
			t.Fatalf("event %q has empty SourcePath", event.SessionID)
		}
		if event.EventID == "" {
			t.Fatalf("event %q has empty EventID", event.SessionID)
		}
	}
}

func TestCollectUsageEvents_DefaultRootAndExplicitDirRules(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := filepath.Join(home, ".codetok", "cursor")
	writeCursorCSVFixture(t, filepath.Join(root, "legacy.csv"),
		`"2026-02-17T10:00:00Z","Included","legacy-model","1","2","3","4"`,
	)
	writeCursorCSVFixture(t, filepath.Join(root, "imports", "manual.csv"),
		`"2026-02-18T10:00:00Z","Included","manual-model","5","6","7","8"`,
	)
	writeCursorCSVFixture(t, filepath.Join(root, "synced", "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","9","10","11","12"`,
	)
	writeCursorCSVFixture(t, filepath.Join(root, "archive", "ignored.csv"),
		`"2026-02-20T10:00:00Z","Included","archived-model","13","14","15","16"`,
	)

	explicitDir := t.TempDir()
	writeCursorCSVFixture(t, filepath.Join(explicitDir, "custom.csv"),
		`"2026-02-21T10:00:00Z","Included","custom-model","17","18","19","20"`,
	)

	p := &Provider{}
	defaultEvents, err := p.CollectUsageEvents("")
	if err != nil {
		t.Fatalf("default root unexpected error: %v", err)
	}
	if len(defaultEvents) != 3 {
		t.Fatalf("default root got %d events, want 3", len(defaultEvents))
	}
	gotDefaultIDs := []string{defaultEvents[0].SessionID, defaultEvents[1].SessionID, defaultEvents[2].SessionID}
	wantDefaultIDs := []string{"legacy:1", "manual:1", "cached:1"}
	for i := range wantDefaultIDs {
		if gotDefaultIDs[i] != wantDefaultIDs[i] {
			t.Fatalf("default sessionIDs[%d] = %q, want %q", i, gotDefaultIDs[i], wantDefaultIDs[i])
		}
	}

	explicitEvents, err := p.CollectUsageEvents(explicitDir)
	if err != nil {
		t.Fatalf("explicit dir unexpected error: %v", err)
	}
	if len(explicitEvents) != 1 {
		t.Fatalf("explicit dir got %d events, want 1", len(explicitEvents))
	}
	if explicitEvents[0].SessionID != "custom:1" {
		t.Fatalf("explicit SessionID = %q, want custom:1", explicitEvents[0].SessionID)
	}
}

func TestCollectUsageEventsInRange_FiltersCSVRowsWithoutUsingFileModTime(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "usage.csv")
	writeCursorCSVFixture(t, path,
		`"2026-04-15T23:59:00Z","before","cursor-auto","0","100","0","10"`,
		`"2026-04-16T12:00:00Z","inside","cursor-auto","0","200","0","20"`,
		`"2026-04-17T00:00:00Z","after","cursor-auto","0","300","0","30"`,
	)
	oldModTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, oldModTime, oldModTime); err != nil {
		t.Fatal(err)
	}

	var metrics provider.UsageEventCollectMetrics
	events, err := (&Provider{}).CollectUsageEventsInRange(baseDir, provider.UsageEventCollectOptions{
		Since:    time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		Until:    time.Date(2026, 4, 16, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
		Location: time.UTC,
		Metrics:  &metrics,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want only in-range row: %#v", len(events), events)
	}
	if events[0].SessionID != "usage:2" || events[0].TokenUsage.Total() != 220 {
		t.Fatalf("event = %#v, want usage:2 total 220", events[0])
	}
	if metrics.ConsideredFiles != 1 || metrics.SkippedFiles != 0 || metrics.ParsedFiles != 1 || metrics.EmittedEvents != 1 {
		t.Fatalf("metrics = %+v, want considered=1 skipped=0 parsed=1 emitted=1", metrics)
	}
}

func TestCollectUsageEventsInRange_ExplicitDirRules(t *testing.T) {
	defaultRoot := t.TempDir()
	explicitDir := t.TempDir()
	t.Setenv("HOME", defaultRoot)
	writeCursorCSVFixture(t, filepath.Join(defaultRoot, ".codetok", "cursor", "imports", "default.csv"),
		`"2026-04-16T10:00:00Z","Default","cursor-auto","0","100","0","10"`,
	)
	writeCursorCSVFixture(t, filepath.Join(explicitDir, "custom.csv"),
		`"2026-04-16T11:00:00Z","Custom","cursor-auto","0","200","0","20"`,
	)

	events, err := (&Provider{}).CollectUsageEventsInRange(explicitDir, provider.UsageEventCollectOptions{
		Since:    time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		Until:    time.Date(2026, 4, 16, 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC),
		Location: time.UTC,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].SessionID != "custom:1" {
		t.Fatalf("events = %#v, want only explicit custom csv", events)
	}
}

func TestParseUsageCSV_SkipsMalformedRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.csv")
	content := `Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"not-a-date","Included","auto","No","10","20","30","40","100","0.01"
"2026-02-18T11:30:00.000Z","On-Demand","gpt-5-codex","No","0","8263","66964","1612","76839","0.03"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, err := parseUsageCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].SessionID != "cursor:2" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "cursor:2")
	}
	if sessions[0].TokenUsage.Total() != 76839 {
		t.Fatalf("Total = %d, want 76839", sessions[0].TokenUsage.Total())
	}
}

func TestParseUsageCSV_SkipsRowsWithWrongFieldCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor.csv")
	content := `Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2026-02-17T10:00:00.000Z","Included","auto","No","28342","775","105891","21282","156290","0.19"
"2026-02-18T11:30:00.000Z","On-Demand","gpt-5-codex","No","0","8263"
"2026-02-19T09:15:00.000Z","Included","claude-3-7-sonnet","No","12","34","56","78","180","0.02"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, err := parseUsageCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].SessionID != "cursor:1" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "cursor:1")
	}
	if sessions[1].SessionID != "cursor:3" {
		t.Fatalf("SessionID = %q, want %q", sessions[1].SessionID, "cursor:3")
	}
	if sessions[1].TokenUsage.Total() != 180 {
		t.Fatalf("Total = %d, want 180", sessions[1].TokenUsage.Total())
	}
}

func TestParseUsageCSV_LegacyExportWithoutKindColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.csv")
	content := `Date,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2026-02-20T08:45:00Z","gpt-4.1","123","456","789","10","999999","12.34"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, err := parseUsageCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}

	session := sessions[0]
	if session.SessionID != "legacy:1" {
		t.Fatalf("SessionID = %q, want %q", session.SessionID, "legacy:1")
	}
	if session.Title != "gpt-4.1" {
		t.Fatalf("Title = %q, want %q", session.Title, "gpt-4.1")
	}
	if session.TokenUsage.InputCacheCreate != 123 {
		t.Fatalf("InputCacheCreate = %d, want 123", session.TokenUsage.InputCacheCreate)
	}
	if session.TokenUsage.InputOther != 456 {
		t.Fatalf("InputOther = %d, want 456", session.TokenUsage.InputOther)
	}
	if session.TokenUsage.InputCacheRead != 789 {
		t.Fatalf("InputCacheRead = %d, want 789", session.TokenUsage.InputCacheRead)
	}
	if session.TokenUsage.Output != 10 {
		t.Fatalf("Output = %d, want 10", session.TokenUsage.Output)
	}
	if session.TokenUsage.Total() != 1378 {
		t.Fatalf("Total = %d, want 1378", session.TokenUsage.Total())
	}
}

func TestParseUsageCSV_BlankTokenCellsMapToZeroWithoutUsingTotalTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blank-tokens.csv")
	content := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2026-02-21T09:15:00Z","Included","claude-3-7-sonnet","","34","","56","999999","88.88"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, err := parseUsageCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}

	usage := sessions[0].TokenUsage
	if usage.InputCacheCreate != 0 {
		t.Fatalf("InputCacheCreate = %d, want 0", usage.InputCacheCreate)
	}
	if usage.InputOther != 34 {
		t.Fatalf("InputOther = %d, want 34", usage.InputOther)
	}
	if usage.InputCacheRead != 0 {
		t.Fatalf("InputCacheRead = %d, want 0", usage.InputCacheRead)
	}
	if usage.Output != 56 {
		t.Fatalf("Output = %d, want 56", usage.Output)
	}
	if usage.Total() != 90 {
		t.Fatalf("Total = %d, want 90", usage.Total())
	}
}

func TestParseUsageCSV_AcceptsRFC3339TimestampVariants(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "time-formats.csv")
	content := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens
"2026-02-20T08:45:00Z","Included","auto","1","2","3","4"
"2026-02-20T08:45:00+08:00","Included","gpt-5-codex","5","6","7","8"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, err := parseUsageCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	if got, want := sessions[0].StartTime.Format(time.RFC3339), "2026-02-20T08:45:00Z"; got != want {
		t.Fatalf("first StartTime = %q, want %q", got, want)
	}
	if got, want := sessions[1].StartTime.Format(time.RFC3339), "2026-02-20T08:45:00+08:00"; got != want {
		t.Fatalf("second StartTime = %q, want %q", got, want)
	}
}

func TestCollectSessions_MultipleCSVFiles(t *testing.T) {
	baseDir := t.TempDir()

	first := `Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2026-02-17T10:00:00.000Z","Included","auto","No","28342","775","105891","21282","156290","0.19"
`
	second := `Date,Kind,Model,Max Mode,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens,Total Tokens,Cost
"2026-02-18T11:30:00.000Z","On-Demand","gpt-5-codex","No","0","8263","66964","1612","76839","0.03"
"2026-02-19T09:15:00.000Z","Included","claude-3-7-sonnet","No","12","34","56","78","180","0.02"
`

	if err := os.WriteFile(filepath.Join(baseDir, "usage-a.csv"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "usage-b.csv"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	sessions, err := p.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	for _, session := range sessions {
		if session.ProviderName != "cursor" {
			t.Fatalf("ProviderName = %q, want %q", session.ProviderName, "cursor")
		}
	}
}

func TestCollectSessions_SkipsInvalidCSVFileAndKeepsDeterministicOrder(t *testing.T) {
	baseDir := t.TempDir()

	invalid := `Date,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read
"2026-02-17T10:00:00Z","broken","1","2","3"
`
	first := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens
"2026-02-21T09:15:00Z","Included","model-b","1","2","3","4"
`
	second := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens
"2026-02-20T09:15:00Z","Included","model-a","5","6","7","8"
"2026-02-21T09:15:00Z","Included","model-c","9","10","11","12"
`

	if err := os.WriteFile(filepath.Join(baseDir, "zzz-invalid.csv"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "bbb.csv"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "aaa.csv"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	sessions, err := p.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	gotIDs := []string{sessions[0].SessionID, sessions[1].SessionID, sessions[2].SessionID}
	wantIDs := []string{"aaa:1", "aaa:2", "bbb:1"}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("sessionIDs[%d] = %q, want %q", i, gotIDs[i], wantIDs[i])
		}
	}
}

func TestCollectSessions_DefaultRootMergesLegacyImportsAndSyncedOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := filepath.Join(home, ".codetok", "cursor")
	writeCursorCSVFixture(t, filepath.Join(root, "legacy.csv"),
		`"2026-02-17T10:00:00Z","Included","legacy-model","1","2","3","4"`,
	)
	writeCursorCSVFixture(t, filepath.Join(root, "imports", "manual.csv"),
		`"2026-02-18T10:00:00Z","Included","manual-model","5","6","7","8"`,
	)
	writeCursorCSVFixture(t, filepath.Join(root, "synced", "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","9","10","11","12"`,
	)
	writeCursorCSVFixture(t, filepath.Join(root, "archive", "ignored.csv"),
		`"2026-02-20T10:00:00Z","Included","archived-model","13","14","15","16"`,
	)

	p := &Provider{}
	sessions, err := p.CollectSessions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	gotIDs := []string{sessions[0].SessionID, sessions[1].SessionID, sessions[2].SessionID}
	wantIDs := []string{"legacy:1", "manual:1", "cached:1"}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("sessionIDs[%d] = %q, want %q", i, gotIDs[i], wantIDs[i])
		}
	}
}

func TestCollectSessions_DefaultRootAllowsSingleSource(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := filepath.Join(home, ".codetok", "cursor")
	writeCursorCSVFixture(t, filepath.Join(root, "synced", "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","9","10","11","12"`,
	)

	p := &Provider{}
	sessions, err := p.CollectSessions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].SessionID != "cached:1" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "cached:1")
	}
}

func TestCollectSessions_ExplicitDirUsesOnlyProvidedDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultRoot := filepath.Join(home, ".codetok", "cursor")
	writeCursorCSVFixture(t, filepath.Join(defaultRoot, "imports", "default.csv"),
		`"2026-02-17T10:00:00Z","Included","default-model","1","2","3","4"`,
	)

	explicitDir := t.TempDir()
	writeCursorCSVFixture(t, filepath.Join(explicitDir, "custom.csv"),
		`"2026-02-21T10:00:00Z","Included","custom-model","5","6","7","8"`,
	)

	p := &Provider{}
	sessions, err := p.CollectSessions(explicitDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].SessionID != "custom:1" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "custom:1")
	}
}

func TestCollectSessions_FindsSyncedCSVInNestedDirectory(t *testing.T) {
	baseDir := t.TempDir()
	syncedDir := filepath.Join(baseDir, "synced")
	if err := os.MkdirAll(syncedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens
"2026-03-15T09:15:00Z","Included","cursor-auto","1","2","3","4"
`
	if err := os.WriteFile(filepath.Join(syncedDir, "usage.csv"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	sessions, err := p.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].SessionID != "usage:1" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "usage:1")
	}
}

func writeCursorCSVFixture(t *testing.T, path string, rows ...string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating parent directory: %v", err)
	}

	content := "Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens\n" +
		rows[0]
	for _, row := range rows[1:] {
		content += "\n" + row
	}
	content += "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing csv fixture: %v", err)
	}
}
