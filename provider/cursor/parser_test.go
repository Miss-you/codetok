package cursor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
