package claude

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

func TestParseClaudeSession_ValidData(t *testing.T) {
	info, err := parseSession(filepath.Join("testdata", "project-a", "session-1.jsonl"), "project-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 assistant messages with usage:
	// input_tokens: 100+200+50 = 350
	// cache_creation_input_tokens: 50+100+25 = 175
	// cache_read_input_tokens: 200+400+100 = 700
	// output_tokens: 30+60+15 = 105
	if info.TokenUsage.InputOther != 350 {
		t.Errorf("InputOther = %d, want 350", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.InputCacheCreate != 175 {
		t.Errorf("InputCacheCreate = %d, want 175", info.TokenUsage.InputCacheCreate)
	}
	if info.TokenUsage.InputCacheRead != 700 {
		t.Errorf("InputCacheRead = %d, want 700", info.TokenUsage.InputCacheRead)
	}
	if info.TokenUsage.Output != 105 {
		t.Errorf("Output = %d, want 105", info.TokenUsage.Output)
	}

	if info.TokenUsage.TotalInput() != 1225 {
		t.Errorf("TotalInput = %d, want 1225", info.TokenUsage.TotalInput())
	}
	if info.TokenUsage.Total() != 1330 {
		t.Errorf("Total = %d, want 1330", info.TokenUsage.Total())
	}

	// 3 user messages (all external)
	if info.Turns != 3 {
		t.Errorf("Turns = %d, want 3", info.Turns)
	}

	if info.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want %q", info.SessionID, "session-1")
	}

	if info.ProviderName != "claude" {
		t.Errorf("ProviderName = %q, want %q", info.ProviderName, "claude")
	}
	if info.ModelName != "claude-opus-4-6" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "claude-opus-4-6")
	}

	if info.WorkDirHash != "project-a" {
		t.Errorf("WorkDirHash = %q, want %q", info.WorkDirHash, "project-a")
	}

	// Title should be first user message text
	if info.Title != "Hello world" {
		t.Errorf("Title = %q, want %q", info.Title, "Hello world")
	}

	if info.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
	if info.EndTime.IsZero() {
		t.Error("EndTime should not be zero")
	}
	if !info.EndTime.After(info.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

func TestParseClaudeSession_Dedup(t *testing.T) {
	dir := t.TempDir()
	// Simulate streaming: the same messageId:requestId appears 3 times with
	// increasing token counts. Only the last entry should be counted.
	// Also include a second unique message to verify independent messages sum correctly.
	content := `{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-02-15T10:00:00.000Z","message":{"id":"msg-A","role":"assistant","content":[{"type":"text","text":"partial 1"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":5}}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-02-15T10:00:01.000Z","message":{"id":"msg-A","role":"assistant","content":[{"type":"text","text":"partial 2"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":15}}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-02-15T10:00:02.000Z","message":{"id":"msg-A","role":"assistant","content":[{"type":"text","text":"final"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":30}}}
{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-02-15T10:01:00.000Z","message":{"role":"user","content":"More"}}
{"type":"assistant","requestId":"req-B","sessionId":"s1","timestamp":"2026-02-15T10:02:00.000Z","message":{"id":"msg-B","role":"assistant","content":[{"type":"text","text":"second reply"}],"usage":{"input_tokens":200,"cache_creation_input_tokens":20,"cache_read_input_tokens":300,"output_tokens":40}}}
`
	sessionPath := filepath.Join(dir, "dedup.jsonl")
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// msg-A:req-A final entry: input=50, cacheCreate=10, cacheRead=100, output=30
	// msg-B:req-B: input=200, cacheCreate=20, cacheRead=300, output=40
	// Totals: input=250, cacheCreate=30, cacheRead=400, output=70

	if info.TokenUsage.InputOther != 250 {
		t.Errorf("InputOther = %d, want 250", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.InputCacheCreate != 30 {
		t.Errorf("InputCacheCreate = %d, want 30", info.TokenUsage.InputCacheCreate)
	}
	if info.TokenUsage.InputCacheRead != 400 {
		t.Errorf("InputCacheRead = %d, want 400", info.TokenUsage.InputCacheRead)
	}
	if info.TokenUsage.Output != 70 {
		t.Errorf("Output = %d, want 70", info.TokenUsage.Output)
	}

	// Without dedup, output would be 5+15+30+40 = 90
	// With dedup, it should be 30+40 = 70
	if info.TokenUsage.Total() != 750 {
		t.Errorf("Total = %d, want 750", info.TokenUsage.Total())
	}

	if info.Turns != 2 {
		t.Errorf("Turns = %d, want 2", info.Turns)
	}
	if info.ModelName != "" {
		t.Errorf("ModelName = %q, want empty", info.ModelName)
	}
}

func TestParseClaudeSession_Dedup_NoIDs(t *testing.T) {
	dir := t.TempDir()
	// Entries without messageId or requestId should each be treated as unique
	content := `{"type":"assistant","sessionId":"s1","timestamp":"2026-02-15T10:00:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"a"}],"usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":5}}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-02-15T10:01:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"b"}],"usage":{"input_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":10}}}
`
	sessionPath := filepath.Join(dir, "no-ids.jsonl")
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both entries should be counted (no dedup since no IDs)
	if info.TokenUsage.InputOther != 30 {
		t.Errorf("InputOther = %d, want 30", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.Output != 15 {
		t.Errorf("Output = %d, want 15", info.TokenUsage.Output)
	}
}

func TestParseClaudeSession_Dedup_PartialIDs(t *testing.T) {
	dir := t.TempDir()
	// Test edge cases:
	// 1. Only messageId present (no requestId) — same messageId should dedup
	// 2. Only requestId present (no messageId) — same requestId should dedup
	// 3. Same requestId but different messageId — should NOT dedup (different keys)
	content := `{"type":"assistant","requestId":"","sessionId":"s1","timestamp":"2026-02-15T10:00:00.000Z","message":{"id":"msg-X","role":"assistant","content":[{"type":"text","text":"a"}],"usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":5}}}
{"type":"assistant","requestId":"","sessionId":"s1","timestamp":"2026-02-15T10:00:01.000Z","message":{"id":"msg-X","role":"assistant","content":[{"type":"text","text":"a final"}],"usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":20}}}
{"type":"assistant","requestId":"req-Y","sessionId":"s1","timestamp":"2026-02-15T10:01:00.000Z","message":{"id":"","role":"assistant","content":[{"type":"text","text":"b"}],"usage":{"input_tokens":30,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":10}}}
{"type":"assistant","requestId":"req-Y","sessionId":"s1","timestamp":"2026-02-15T10:01:01.000Z","message":{"id":"","role":"assistant","content":[{"type":"text","text":"b final"}],"usage":{"input_tokens":30,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":25}}}
{"type":"assistant","requestId":"req-Z","sessionId":"s1","timestamp":"2026-02-15T10:02:00.000Z","message":{"id":"msg-A","role":"assistant","content":[{"type":"text","text":"c"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":15}}}
{"type":"assistant","requestId":"req-Z","sessionId":"s1","timestamp":"2026-02-15T10:02:01.000Z","message":{"id":"msg-B","role":"assistant","content":[{"type":"text","text":"d"}],"usage":{"input_tokens":60,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":30}}}
`
	sessionPath := filepath.Join(dir, "partial-ids.jsonl")
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key "msg-X:" — two entries, last wins: input=10, output=20
	// Key ":req-Y" — two entries, last wins: input=30, output=25
	// Key "msg-A:req-Z" — one entry: input=50, output=15
	// Key "msg-B:req-Z" — one entry: input=60, output=30
	// (msg-A:req-Z and msg-B:req-Z are distinct keys because messageId differs)
	//
	// Totals: input = 10+30+50+60 = 150, output = 20+25+15+30 = 90
	if info.TokenUsage.InputOther != 150 {
		t.Errorf("InputOther = %d, want 150", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.Output != 90 {
		t.Errorf("Output = %d, want 90", info.TokenUsage.Output)
	}
}

func TestParseClaudeUsageEvents_UsesAssistantTimestampAndMetadata(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")
	content := `{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-04-15T23:50:00Z","message":{"role":"user","content":"Investigate token attribution"}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-04-16T00:10:00Z","message":{"id":"msg-A","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"Done"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":20,"cache_read_input_tokens":30,"output_tokens":40}}}
`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseUsageEvents(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	got := events[0]
	wantAssistantTS := mustParseTime(t, "2026-04-16T00:10:00Z")
	if !got.Timestamp.Equal(wantAssistantTS) {
		t.Errorf("Timestamp = %s, want assistant timestamp %s", got.Timestamp, wantAssistantTS)
	}
	if got.Timestamp.Equal(mustParseTime(t, "2026-04-15T23:50:00Z")) {
		t.Error("Timestamp should not use the user/session start timestamp")
	}
	if got.ProviderName != "claude" {
		t.Errorf("ProviderName = %q, want claude", got.ProviderName)
	}
	if got.SessionID != "s1" {
		t.Errorf("SessionID = %q, want s1", got.SessionID)
	}
	if got.ModelName != "claude-sonnet-4-6" {
		t.Errorf("ModelName = %q, want claude-sonnet-4-6", got.ModelName)
	}
	if got.Title != "Investigate token attribution" {
		t.Errorf("Title = %q, want first user text", got.Title)
	}
	if got.WorkDirHash != "project-x" {
		t.Errorf("WorkDirHash = %q, want project-x", got.WorkDirHash)
	}
	if got.SourcePath != sessionPath {
		t.Errorf("SourcePath = %q, want %q", got.SourcePath, sessionPath)
	}
	if got.EventID != "msg-A:req-A" {
		t.Errorf("EventID = %q, want msg-A:req-A", got.EventID)
	}
	if got.TokenUsage.InputOther != 100 {
		t.Errorf("InputOther = %d, want 100", got.TokenUsage.InputOther)
	}
	if got.TokenUsage.InputCacheCreate != 20 {
		t.Errorf("InputCacheCreate = %d, want 20", got.TokenUsage.InputCacheCreate)
	}
	if got.TokenUsage.InputCacheRead != 30 {
		t.Errorf("InputCacheRead = %d, want 30", got.TokenUsage.InputCacheRead)
	}
	if got.TokenUsage.Output != 40 {
		t.Errorf("Output = %d, want 40", got.TokenUsage.Output)
	}
}

func TestParseClaudeUsageEvents_CrossDayAssistantMessages(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "cross-day.jsonl")
	content := `{"type":"user","userType":"external","sessionId":"cross-day","timestamp":"2026-04-15T23:45:00Z","message":{"role":"user","content":"Continue overnight"}}
{"type":"assistant","requestId":"req-1","sessionId":"cross-day","timestamp":"2026-04-15T23:55:00Z","message":{"id":"msg-1","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"First"}],"usage":{"input_tokens":10,"cache_creation_input_tokens":1,"cache_read_input_tokens":2,"output_tokens":3}}}
{"type":"assistant","requestId":"req-2","sessionId":"cross-day","timestamp":"2026-04-16T00:05:00Z","message":{"id":"msg-2","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"Second"}],"usage":{"input_tokens":20,"cache_creation_input_tokens":4,"cache_read_input_tokens":5,"output_tokens":6}}}
`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseUsageEvents(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	if events[0].SessionID != "cross-day" || events[1].SessionID != "cross-day" {
		t.Fatalf("events should keep the same session ID: %#v", events)
	}
	if !events[0].Timestamp.Equal(mustParseTime(t, "2026-04-15T23:55:00Z")) {
		t.Errorf("first event timestamp = %s, want 2026-04-15T23:55:00Z", events[0].Timestamp)
	}
	if !events[1].Timestamp.Equal(mustParseTime(t, "2026-04-16T00:05:00Z")) {
		t.Errorf("second event timestamp = %s, want 2026-04-16T00:05:00Z", events[1].Timestamp)
	}
	if events[0].TokenUsage.Total() != 16 {
		t.Errorf("first total = %d, want 16", events[0].TokenUsage.Total())
	}
	if events[1].TokenUsage.Total() != 35 {
		t.Errorf("second total = %d, want 35", events[1].TokenUsage.Total())
	}
}

func TestParseClaudeUsageEvents_DedupStreamingKeepsLatestUsage(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "dedup-events.jsonl")
	content := `{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-04-16T09:59:00Z","message":{"role":"user","content":"Stream"}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-04-16T10:00:00Z","message":{"id":"msg-A","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"partial"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":5}}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-04-16T10:00:01Z","message":{"id":"msg-A","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"partial"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":15}}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-04-16T10:00:02Z","message":{"id":"msg-A","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"final"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":30}}}
{"type":"assistant","requestId":"req-B","sessionId":"s1","timestamp":"2026-04-16T10:02:00Z","message":{"id":"msg-B","model":"claude-haiku-4-5","role":"assistant","content":[{"type":"text","text":"second"}],"usage":{"input_tokens":200,"cache_creation_input_tokens":20,"cache_read_input_tokens":300,"output_tokens":40}}}
`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseUsageEvents(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	if events[0].EventID != "msg-A:req-A" {
		t.Fatalf("first EventID = %q, want msg-A:req-A", events[0].EventID)
	}
	if !events[0].Timestamp.Equal(mustParseTime(t, "2026-04-16T10:00:02Z")) {
		t.Errorf("deduped timestamp = %s, want latest streaming timestamp", events[0].Timestamp)
	}
	if events[0].TokenUsage.Output != 30 {
		t.Errorf("deduped Output = %d, want latest 30", events[0].TokenUsage.Output)
	}
	if events[0].TokenUsage.Total() != 190 {
		t.Errorf("deduped total = %d, want 190", events[0].TokenUsage.Total())
	}
	if events[1].EventID != "msg-B:req-B" {
		t.Fatalf("second EventID = %q, want msg-B:req-B", events[1].EventID)
	}
	if events[1].ModelName != "claude-haiku-4-5" {
		t.Errorf("second ModelName = %q, want claude-haiku-4-5", events[1].ModelName)
	}
}

func TestParseClaudeUsageEvents_DedupUsesLatestFileRecord(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "dedup-file-order.jsonl")
	content := `{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-04-16T10:00:02Z","message":{"id":"msg-A","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"first final-looking row"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":100,"output_tokens":5}}}
{"type":"assistant","requestId":"req-A","sessionId":"s1","timestamp":"2026-04-16T10:00:01Z","message":{"id":"msg-A","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"later file row"}],"usage":{"input_tokens":60,"cache_creation_input_tokens":11,"cache_read_input_tokens":101,"output_tokens":30}}}
`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseUsageEvents(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	got := events[0]
	if !got.Timestamp.Equal(mustParseTime(t, "2026-04-16T10:00:01Z")) {
		t.Errorf("Timestamp = %s, want latest file record timestamp", got.Timestamp)
	}
	if got.TokenUsage.InputOther != 60 {
		t.Errorf("InputOther = %d, want latest file record 60", got.TokenUsage.InputOther)
	}
	if got.TokenUsage.InputCacheCreate != 11 {
		t.Errorf("InputCacheCreate = %d, want latest file record 11", got.TokenUsage.InputCacheCreate)
	}
	if got.TokenUsage.InputCacheRead != 101 {
		t.Errorf("InputCacheRead = %d, want latest file record 101", got.TokenUsage.InputCacheRead)
	}
	if got.TokenUsage.Output != 30 {
		t.Errorf("Output = %d, want latest file record 30", got.TokenUsage.Output)
	}
}

func TestParseClaudeUsageEvents_NoIDsRemainUnique(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "no-ids-events.jsonl")
	content := `{"type":"assistant","sessionId":"s1","timestamp":"2026-04-16T10:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"a"}],"usage":{"input_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":5}}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-04-16T10:01:00Z","message":{"role":"assistant","content":[{"type":"text","text":"b"}],"usage":{"input_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":10}}}
`
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseUsageEvents(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want both no-ID entries to stay unique", len(events))
	}
	if events[0].EventID == "" || events[1].EventID == "" || events[0].EventID == events[1].EventID {
		t.Fatalf("events should have distinct non-empty fallback IDs: %#v", events)
	}
	if events[0].TokenUsage.Total() != 15 {
		t.Errorf("first total = %d, want 15", events[0].TokenUsage.Total())
	}
	if events[1].TokenUsage.Total() != 30 {
		t.Errorf("second total = %d, want 30", events[1].TokenUsage.Total())
	}
}

func TestCollectClaudeUsageEvents_IncludesSubagentPaths(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "project-x")
	subagentsDir := filepath.Join(projectDir, "session-abc", "subagents")
	if err := os.MkdirAll(subagentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	parentPath := filepath.Join(projectDir, "session-abc.jsonl")
	parentContent := `{"type":"assistant","requestId":"req-parent","sessionId":"session-abc","timestamp":"2026-04-16T10:00:00Z","message":{"id":"msg-parent","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"parent"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":30}}}
`
	if err := os.WriteFile(parentPath, []byte(parentContent), 0644); err != nil {
		t.Fatal(err)
	}

	subagentPath := filepath.Join(subagentsDir, "agent-a123.jsonl")
	subagentContent := `{"type":"assistant","requestId":"req-sub","sessionId":"session-abc","timestamp":"2026-04-16T10:02:00Z","message":{"id":"msg-sub","model":"claude-haiku-4-5","role":"assistant","content":[{"type":"text","text":"sub"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":20}}}
`
	if err := os.WriteFile(subagentPath, []byte(subagentContent), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := (&Provider{}).CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want parent + subagent", len(events))
	}

	byPath := make(map[string]bool)
	for _, event := range events {
		byPath[event.SourcePath] = true
		if event.WorkDirHash != "project-x" {
			t.Errorf("WorkDirHash = %q, want project-x", event.WorkDirHash)
		}
	}
	if !byPath[parentPath] {
		t.Errorf("missing parent event source %q in %#v", parentPath, events)
	}
	if !byPath[subagentPath] {
		t.Errorf("missing subagent event source %q in %#v", subagentPath, events)
	}
}

func TestCollectClaudeUsageEventsWithParser_ParsesInParallelAndSorts(t *testing.T) {
	paths := []string{"file-c.jsonl", "file-a.jsonl", "bad.jsonl", "file-b.jsonl"}
	pathToSlug := map[string]string{
		"file-a.jsonl": "project-a",
		"file-b.jsonl": "project-b",
		"file-c.jsonl": "project-c",
		"bad.jsonl":    "project-bad",
	}

	var active int64
	var maxActive int64
	started := make(chan string, len(paths))
	release := make(chan struct{})
	parser := func(path, projectSlug string) ([]provider.UsageEvent, error) {
		current := atomic.AddInt64(&active, 1)
		for {
			observed := atomic.LoadInt64(&maxActive)
			if current <= observed || atomic.CompareAndSwapInt64(&maxActive, observed, current) {
				break
			}
		}
		defer atomic.AddInt64(&active, -1)

		started <- path
		<-release
		if path == "bad.jsonl" {
			return nil, errors.New("skip this file")
		}
		return []provider.UsageEvent{{
			ProviderName: "claude",
			WorkDirHash:  projectSlug,
			SourcePath:   path,
			Timestamp:    time.Date(2026, 4, 16, len(path), 0, 0, 0, time.UTC),
			EventID:      path + ":event",
		}}, nil
	}

	resultCh := make(chan []provider.UsageEvent, 1)
	go func() {
		resultCh <- collectUsageEventsWithParser(paths, pathToSlug, 2, parser)
	}()

	startedPaths := make(map[string]bool)
	for len(startedPaths) < 2 {
		select {
		case path := <-started:
			startedPaths[path] = true
		case <-time.After(time.Second):
			close(release)
			t.Fatalf("timed out waiting for two concurrent parser starts; saw %v", startedPaths)
		}
	}
	close(release)

	var events []provider.UsageEvent
	select {
	case events = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for parallel collection to finish")
	}
	if got := atomic.LoadInt64(&maxActive); got < 2 {
		t.Fatalf("max concurrent parses = %d, want bounded parallel parsing with at least 2 active workers", got)
	} else if got > 2 {
		t.Fatalf("max concurrent parses = %d, want worker limit 2", got)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 successful parses", len(events))
	}
	wantPaths := []string{"file-a.jsonl", "file-b.jsonl", "file-c.jsonl"}
	for i, want := range wantPaths {
		if events[i].SourcePath != want {
			t.Fatalf("event[%d].SourcePath = %q, want deterministic source order %q in %#v", i, events[i].SourcePath, want, events)
		}
	}
	if events[0].WorkDirHash != "project-a" || events[1].WorkDirHash != "project-b" || events[2].WorkDirHash != "project-c" {
		t.Fatalf("events did not receive project slugs from path map: %#v", events)
	}
}

func BenchmarkCollectClaudeUsageEventsSynthetic(b *testing.B) {
	const (
		projectCount    = 4
		filesPerProject = 25
		eventsPerFile   = 6
	)
	baseDir := b.TempDir()
	totalFiles := writeSyntheticClaudeUsageTree(b, baseDir, projectCount, filesPerProject, eventsPerFile)
	totalEvents := totalFiles * eventsPerFile
	p := &Provider{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		events, err := p.CollectUsageEvents(baseDir)
		if err != nil {
			b.Fatalf("CollectUsageEvents returned error: %v", err)
		}
		if len(events) != totalEvents {
			b.Fatalf("got %d events, want %d", len(events), totalEvents)
		}
	}
	b.ReportMetric(float64(totalFiles), "files/op")
	b.ReportMetric(float64(totalEvents), "events/op")
}

func writeSyntheticClaudeUsageTree(tb testing.TB, baseDir string, projectCount, filesPerProject, eventsPerFile int) int {
	tb.Helper()

	totalFiles := 0
	for projectIndex := 0; projectIndex < projectCount; projectIndex++ {
		projectDir := filepath.Join(baseDir, fmt.Sprintf("project-%02d", projectIndex))
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			tb.Fatal(err)
		}
		for fileIndex := 0; fileIndex < filesPerProject; fileIndex++ {
			sessionID := fmt.Sprintf("session-%02d-%02d", projectIndex, fileIndex)
			path := filepath.Join(projectDir, sessionID+".jsonl")
			var content strings.Builder
			for eventIndex := 0; eventIndex < eventsPerFile; eventIndex++ {
				timestamp := time.Date(2026, 4, 10+projectIndex, fileIndex%24, eventIndex, 0, 0, time.UTC).Format(time.RFC3339)
				fmt.Fprintf(&content, `{"type":"assistant","requestId":"req-%02d-%02d-%02d","sessionId":"%s","timestamp":"%s","message":{"id":"msg-%02d-%02d-%02d","model":"claude-sonnet-4-6","role":"assistant","content":[{"type":"text","text":"synthetic"}],"usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d}}}`+"\n",
					projectIndex,
					fileIndex,
					eventIndex,
					sessionID,
					timestamp,
					projectIndex,
					fileIndex,
					eventIndex,
					100+eventIndex,
					10,
					20,
					30+eventIndex,
				)
			}
			if err := os.WriteFile(path, []byte(content.String()), 0644); err != nil {
				tb.Fatal(err)
			}
			totalFiles++
		}
	}
	return totalFiles
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return ts
}

func TestParseClaudeSession_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(emptyFile, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenUsage.Total() != 0 {
		t.Errorf("Total = %d, want 0", info.TokenUsage.Total())
	}
	if info.Turns != 0 {
		t.Errorf("Turns = %d, want 0", info.Turns)
	}
	// Should fall back to filename as session ID
	if info.SessionID != "empty" {
		t.Errorf("SessionID = %q, want %q", info.SessionID, "empty")
	}
}

func TestParseClaudeSession_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
this is not valid json at all
{"type":"assistant","sessionId":"s1","timestamp":"2026-02-15T10:00:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":50,"cache_read_input_tokens":200,"output_tokens":30}}}
`
	sessionPath := filepath.Join(dir, "malformed.jsonl")
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have parsed the one valid assistant message
	if info.TokenUsage.InputOther != 100 {
		t.Errorf("InputOther = %d, want 100", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.Output != 30 {
		t.Errorf("Output = %d, want 30", info.TokenUsage.Output)
	}
	// Should have counted the one valid user message
	if info.Turns != 1 {
		t.Errorf("Turns = %d, want 1", info.Turns)
	}
}

func TestParseClaudeSession_NoAssistantMessages(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-02-15T10:01:00.000Z","message":{"role":"user","content":"Anyone there?"}}
`
	sessionPath := filepath.Join(dir, "no-assistant.jsonl")
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionPath, "project-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenUsage.Total() != 0 {
		t.Errorf("Total = %d, want 0", info.TokenUsage.Total())
	}
	if info.Turns != 2 {
		t.Errorf("Turns = %d, want 2", info.Turns)
	}
}

func TestCollectClaudeSessions_SubagentDiscovery(t *testing.T) {
	baseDir := t.TempDir()

	// Create a project with a top-level session file and subagent files
	projectDir := filepath.Join(baseDir, "project-x")
	subagentsDir := filepath.Join(projectDir, "session-abc", "subagents")
	if err := os.MkdirAll(subagentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionContent := `{"type":"user","userType":"external","sessionId":"session-abc","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","sessionId":"session-abc","timestamp":"2026-02-15T10:00:00.000Z","message":{"id":"msg-parent","role":"assistant","model":"claude-sonnet-4-6","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":50,"cache_read_input_tokens":200,"output_tokens":30}}}
`
	subagentContent := `{"type":"user","userType":"external","sessionId":"session-abc","timestamp":"2026-02-15T10:01:00.000Z","message":{"role":"user","content":"Sub task"}}
{"type":"assistant","sessionId":"session-abc","timestamp":"2026-02-15T10:02:00.000Z","message":{"id":"msg-sub1","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"Done"}],"usage":{"input_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":80,"output_tokens":20}}}
`

	// Write parent session file
	if err := os.WriteFile(filepath.Join(projectDir, "session-abc.jsonl"), []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Write subagent file
	if err := os.WriteFile(filepath.Join(subagentsDir, "agent-a123.jsonl"), []byte(subagentContent), 0644); err != nil {
		t.Fatal(err)
	}

	prov := &Provider{}
	result, err := prov.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("got %d sessions, want 2 (parent + subagent)", len(result))
	}

	// Both should have the same project slug
	for _, s := range result {
		if s.WorkDirHash != "project-x" {
			t.Errorf("WorkDirHash = %q, want %q", s.WorkDirHash, "project-x")
		}
	}

	// Sum up tokens from all sessions
	totalInput := 0
	for _, s := range result {
		totalInput += s.TokenUsage.InputOther
	}
	if totalInput != 150 {
		t.Errorf("total InputOther = %d, want 150 (100 + 50)", totalInput)
	}
}

func TestCollectClaudeSessions_EmptySubagentsDir(t *testing.T) {
	baseDir := t.TempDir()

	projectDir := filepath.Join(baseDir, "project-y")
	subagentsDir := filepath.Join(projectDir, "session-def", "subagents")
	if err := os.MkdirAll(subagentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionContent := `{"type":"user","userType":"external","sessionId":"session-def","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","sessionId":"session-def","timestamp":"2026-02-15T10:00:00.000Z","message":{"id":"msg-1","role":"assistant","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":30}}}
`
	if err := os.WriteFile(filepath.Join(projectDir, "session-def.jsonl"), []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Empty subagents dir — should not cause errors

	prov := &Provider{}
	result, err := prov.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the parent session should be found
	if len(result) != 1 {
		t.Fatalf("got %d sessions, want 1", len(result))
	}
}

func TestCollectClaudeSessions_ExplicitMissingDirReturnsError(t *testing.T) {
	// Explicit --claude-dir pointing to non-existent path should return an error
	baseDir := filepath.Join(t.TempDir(), "nonexistent")

	prov := &Provider{}
	_, err := prov.CollectSessions(baseDir)
	if err == nil {
		t.Fatal("expected error for explicit non-existent directory, got nil")
	}
}

func TestCollectPaths_MissingDirReturnsError(t *testing.T) {
	// collectPaths should return an error for non-existent directories
	baseDir := filepath.Join(t.TempDir(), "nonexistent")

	var paths []string
	pathToSlug := make(map[string]string)
	err := collectPaths(baseDir, &paths, pathToSlug)
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
	if len(paths) != 0 {
		t.Errorf("got %d paths, want 0", len(paths))
	}
}

func TestCollectClaudeSessions_FlagOverride(t *testing.T) {
	// When baseDir is explicitly set, only that directory should be scanned
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "project-z")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionContent := `{"type":"user","userType":"external","sessionId":"s1","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","sessionId":"s1","timestamp":"2026-02-15T10:00:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":30}}}
`
	if err := os.WriteFile(filepath.Join(projectDir, "sess.jsonl"), []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}

	prov := &Provider{}
	result, err := prov.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d sessions, want 1", len(result))
	}
	if result[0].WorkDirHash != "project-z" {
		t.Errorf("WorkDirHash = %q, want %q", result[0].WorkDirHash, "project-z")
	}
}

func TestCollectPaths_MultipleBaseDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	sessionContent := `{"type":"assistant","sessionId":"s1","timestamp":"2026-02-15T10:00:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":30}}}
`

	// Create session in dir1
	if err := os.MkdirAll(filepath.Join(dir1, "proj-a"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "proj-a", "s1.jsonl"), []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create session in dir2
	if err := os.MkdirAll(filepath.Join(dir2, "proj-b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "proj-b", "s2.jsonl"), []byte(sessionContent), 0644); err != nil {
		t.Fatal(err)
	}

	var paths []string
	pathToSlug := make(map[string]string)
	if err := collectPaths(dir1, &paths, pathToSlug); err != nil {
		t.Fatal(err)
	}
	if err := collectPaths(dir2, &paths, pathToSlug); err != nil {
		t.Fatal(err)
	}

	if len(paths) != 2 {
		t.Fatalf("got %d paths, want 2", len(paths))
	}

	slugs := make(map[string]bool)
	for _, p := range paths {
		slugs[pathToSlug[p]] = true
	}
	if !slugs["proj-a"] || !slugs["proj-b"] {
		t.Errorf("expected slugs proj-a and proj-b, got %v", slugs)
	}
}

func TestCollectClaudeSessions_MultipleProjects(t *testing.T) {
	baseDir := t.TempDir()

	// Create two projects with session files
	projects := []struct {
		slug    string
		session string
	}{
		{"project-alpha", "sess-aaa.jsonl"},
		{"project-alpha", "sess-bbb.jsonl"},
		{"project-beta", "sess-ccc.jsonl"},
	}

	sessionContent := `{"type":"user","userType":"external","sessionId":"test-session","timestamp":"2026-02-15T09:59:00.000Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","sessionId":"test-session","timestamp":"2026-02-15T10:00:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":100,"cache_creation_input_tokens":50,"cache_read_input_tokens":200,"output_tokens":30}}}
`

	for _, p := range projects {
		dir := filepath.Join(baseDir, p.slug)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p.session), []byte(sessionContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Also create a non-JSONL file that should be ignored
	if err := os.WriteFile(filepath.Join(baseDir, "project-alpha", "notes.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	// Also create a subdirectory (like metadata/) that should be skipped
	if err := os.MkdirAll(filepath.Join(baseDir, "project-alpha", "metadata"), 0755); err != nil {
		t.Fatal(err)
	}

	prov := &Provider{}
	result, err := prov.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("got %d sessions, want 3", len(result))
	}

	// Check that all sessions have correct provider and token data
	for _, s := range result {
		if s.ProviderName != "claude" {
			t.Errorf("ProviderName = %q, want %q", s.ProviderName, "claude")
		}
		if s.TokenUsage.InputOther != 100 {
			t.Errorf("session %s InputOther = %d, want 100", s.SessionID, s.TokenUsage.InputOther)
		}
		if s.Turns != 1 {
			t.Errorf("session %s Turns = %d, want 1", s.SessionID, s.Turns)
		}
	}

	// Verify project slug is used as WorkDirHash
	slugs := make(map[string]int)
	for _, s := range result {
		slugs[s.WorkDirHash]++
	}
	if slugs["project-alpha"] != 2 {
		t.Errorf("project-alpha sessions = %d, want 2", slugs["project-alpha"])
	}
	if slugs["project-beta"] != 1 {
		t.Errorf("project-beta sessions = %d, want 1", slugs["project-beta"])
	}
}
