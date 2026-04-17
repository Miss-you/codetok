package codex

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/miss-you/codetok/stats"
)

func TestParseCodexSession_ValidData(t *testing.T) {
	info, err := parseCodexSession(filepath.Join("testdata", "2026", "02", "15", "rollout-2026-02-15T10-00-00-test-uuid-1.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.SessionID != "test-uuid-1" {
		t.Errorf("SessionID = %q, want %q", info.SessionID, "test-uuid-1")
	}
	if info.Title != "Hello world" {
		t.Errorf("Title = %q, want %q", info.Title, "Hello world")
	}
	if info.ProviderName != "codex" {
		t.Errorf("ProviderName = %q, want %q", info.ProviderName, "codex")
	}

	// Last token_count: input_tokens=1500, cached_input_tokens=800, output_tokens=300
	// InputOther = 1500 - 800 = 700
	if info.TokenUsage.InputOther != 700 {
		t.Errorf("InputOther = %d, want 700", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.InputCacheRead != 800 {
		t.Errorf("InputCacheRead = %d, want 800", info.TokenUsage.InputCacheRead)
	}
	if info.TokenUsage.Output != 300 {
		t.Errorf("Output = %d, want 300", info.TokenUsage.Output)
	}
	if info.TokenUsage.InputCacheCreate != 0 {
		t.Errorf("InputCacheCreate = %d, want 0", info.TokenUsage.InputCacheCreate)
	}

	// 2 user_message events
	if info.Turns != 2 {
		t.Errorf("Turns = %d, want 2", info.Turns)
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

func TestParseCodexSession_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "rollout-empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(emptyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenUsage.Total() != 0 {
		t.Errorf("Total = %d, want 0", info.TokenUsage.Total())
	}
	if info.Turns != 0 {
		t.Errorf("Turns = %d, want 0", info.Turns)
	}
}

func TestParseCodexSession_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"mal-test","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test","cli_version":"0.47.0"}}
this is not valid json
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":500,"cached_input_tokens":200,"output_tokens":100,"reasoning_output_tokens":20,"total_tokens":600}}}}
`
	filePath := filepath.Join(dir, "rollout-malformed.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SessionID != "mal-test" {
		t.Errorf("SessionID = %q, want %q", info.SessionID, "mal-test")
	}
	// InputOther = 500 - 200 = 300
	if info.TokenUsage.InputOther != 300 {
		t.Errorf("InputOther = %d, want 300", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.Output != 100 {
		t.Errorf("Output = %d, want 100", info.TokenUsage.Output)
	}
}

func TestParseCodexSession_NoTokenCount(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"no-tokens","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test","cli_version":"0.47.0"}}
{"timestamp":"2026-02-15T10:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"Hello"}}
`
	filePath := filepath.Join(dir, "rollout-no-tokens.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenUsage.Total() != 0 {
		t.Errorf("Total = %d, want 0", info.TokenUsage.Total())
	}
	if info.Turns != 1 {
		t.Errorf("Turns = %d, want 1", info.Turns)
	}
	if info.ModelName != "" {
		t.Errorf("ModelName = %q, want empty", info.ModelName)
	}
}

func TestParseCodexSession_ModelExtraction(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"model-test","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test"}}
{"timestamp":"2026-02-15T10:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"Hello"}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"model_name":"gpt-5-codex","total_token_usage":{"input_tokens":500,"cached_input_tokens":200,"output_tokens":100,"total_tokens":600}}}}
`
	filePath := filepath.Join(dir, "rollout-model.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "gpt-5-codex" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "gpt-5-codex")
	}
}

func TestParseCodexSession_ModelExtractionFromNonEventMsgPayload(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"turn-context-model","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test"}}
{"timestamp":"2026-02-15T10:00:01.000Z","type":"turn_context","payload":{"context":{"model_name":"gpt-5-codex"}}}
`
	filePath := filepath.Join(dir, "rollout-turn-context-model.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "gpt-5-codex" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "gpt-5-codex")
	}
}

func TestParseCodexSession_ModelExtractionMalformedInfo(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"model-malformed","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test"}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":"not-an-object"}}
`
	filePath := filepath.Join(dir, "rollout-model-malformed.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "" {
		t.Errorf("ModelName = %q, want empty", info.ModelName)
	}
}

func TestParseCodexSession_ModelExtractionPrefersKnownModelPath(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"model-known-path","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test"}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"limit_name":"rate-limit-tier","context":{"model_name":"gpt-5-codex"},"total_token_usage":{"input_tokens":500,"cached_input_tokens":200,"output_tokens":100,"total_tokens":600}}}}
`
	filePath := filepath.Join(dir, "rollout-model-known-path.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "gpt-5-codex" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "gpt-5-codex")
	}
}

func TestParseCodexSession_ModelExtractionRejectsPlaceholder(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"model-placeholder","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test"}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"model":"default","total_token_usage":{"input_tokens":500,"cached_input_tokens":200,"output_tokens":100,"total_tokens":600}}}}
`
	filePath := filepath.Join(dir, "rollout-model-placeholder.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "" {
		t.Errorf("ModelName = %q, want empty", info.ModelName)
	}
}

func TestParseCodexSession_ModelExtractionSkipsPlaceholderMsgModel(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"model-placeholder-msg","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test"}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","model":"default","info":{"context":{"model_name":"gpt-5-codex"},"total_token_usage":{"input_tokens":500,"cached_input_tokens":200,"output_tokens":100,"total_tokens":600}}}}
`
	filePath := filepath.Join(dir, "rollout-model-placeholder-msg.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "gpt-5-codex" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "gpt-5-codex")
	}
}

func TestParseCodexSession_MultipleTokenCounts(t *testing.T) {
	dir := t.TempDir()
	content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"multi-tc","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test","cli_version":"0.47.0"}}
{"timestamp":"2026-02-15T10:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"First"}}
{"timestamp":"2026-02-15T10:00:02.000Z","type":"event_msg","payload":{"type":"token_count","info":null}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":500,"cached_input_tokens":200,"output_tokens":100,"reasoning_output_tokens":20,"total_tokens":600}}}}
{"timestamp":"2026-02-15T10:01:30.000Z","type":"event_msg","payload":{"type":"user_message","message":"Second"}}
{"timestamp":"2026-02-15T10:02:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":3000,"cached_input_tokens":2000,"output_tokens":800,"reasoning_output_tokens":100,"total_tokens":3800}}}}
`
	filePath := filepath.Join(dir, "rollout-multi.jsonl")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should take the LAST token_count (cumulative): input=3000, cached=2000, output=800
	// InputOther = 3000 - 2000 = 1000
	if info.TokenUsage.InputOther != 1000 {
		t.Errorf("InputOther = %d, want 1000", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.InputCacheRead != 2000 {
		t.Errorf("InputCacheRead = %d, want 2000", info.TokenUsage.InputCacheRead)
	}
	if info.TokenUsage.Output != 800 {
		t.Errorf("Output = %d, want 800", info.TokenUsage.Output)
	}
	if info.Turns != 2 {
		t.Errorf("Turns = %d, want 2", info.Turns)
	}
}

func TestCollectCodexSessions_DateDirStructure(t *testing.T) {
	// Create a temporary directory tree:
	// baseDir/2026/02/15/rollout-a.jsonl
	// baseDir/2026/02/16/rollout-b.jsonl
	baseDir := t.TempDir()

	type testSession struct {
		year, month, day string
		filename         string
		sessionID        string
		title            string
	}

	sessions := []testSession{
		{"2026", "02", "15", "rollout-2026-02-15T10-00-00-uuid-a.jsonl", "uuid-a", "Session A"},
		{"2026", "02", "16", "rollout-2026-02-16T14-00-00-uuid-b.jsonl", "uuid-b", "Session B"},
		{"2026", "01", "10", "rollout-2026-01-10T09-00-00-uuid-c.jsonl", "uuid-c", "Session C"},
	}

	for _, s := range sessions {
		dir := filepath.Join(baseDir, s.year, s.month, s.day)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		content := `{"timestamp":"2026-02-15T10:00:00.000Z","type":"session_meta","payload":{"id":"` + s.sessionID + `","timestamp":"2026-02-15T10:00:00.000Z","cwd":"/test","cli_version":"0.47.0"}}
{"timestamp":"2026-02-15T10:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"` + s.title + `"}}
{"timestamp":"2026-02-15T10:01:00.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":500,"output_tokens":200,"reasoning_output_tokens":50,"total_tokens":1200}}}}
`
		if err := os.WriteFile(filepath.Join(dir, s.filename), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	p := &Provider{}
	result, err := p.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("got %d sessions, want 3", len(result))
	}

	// Verify all sessions have correct data
	ids := make(map[string]bool)
	for _, s := range result {
		ids[s.SessionID] = true
		if s.ProviderName != "codex" {
			t.Errorf("session %s ProviderName = %q, want %q", s.SessionID, s.ProviderName, "codex")
		}
		// InputOther = 1000 - 500 = 500
		if s.TokenUsage.InputOther != 500 {
			t.Errorf("session %s InputOther = %d, want 500", s.SessionID, s.TokenUsage.InputOther)
		}
		if s.Turns != 1 {
			t.Errorf("session %s Turns = %d, want 1", s.SessionID, s.Turns)
		}
	}
	for _, s := range sessions {
		if !ids[s.sessionID] {
			t.Errorf("missing session %s", s.sessionID)
		}
	}
}

func TestParseCodexUsageEvents_LastTokenUsageEmitsOneEvent(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-last.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"last-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:00:01Z","type":"turn_context","payload":{"model":"gpt-5.4"}}
{"timestamp":"2026-04-15T10:00:02Z","type":"event_msg","payload":{"type":"user_message","message":"first question"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":120,"cached_input_tokens":20,"output_tokens":30,"total_tokens":150},"total_token_usage":{"input_tokens":1000,"cached_input_tokens":500,"output_tokens":300,"total_tokens":1300}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}
	event := events[0]
	if event.ProviderName != "codex" {
		t.Errorf("ProviderName = %q, want codex", event.ProviderName)
	}
	if event.SessionID != "last-session" {
		t.Errorf("SessionID = %q, want last-session", event.SessionID)
	}
	if event.Title != "first question" {
		t.Errorf("Title = %q, want first question", event.Title)
	}
	if event.ModelName != "gpt-5.4" {
		t.Errorf("ModelName = %q, want gpt-5.4", event.ModelName)
	}
	if !event.Timestamp.Equal(time.Date(2026, 4, 15, 10, 1, 0, 0, time.UTC)) {
		t.Errorf("Timestamp = %s, want 2026-04-15T10:01:00Z", event.Timestamp.Format(time.RFC3339))
	}
	if event.TokenUsage.InputOther != 100 {
		t.Errorf("InputOther = %d, want 100", event.TokenUsage.InputOther)
	}
	if event.TokenUsage.InputCacheRead != 20 {
		t.Errorf("InputCacheRead = %d, want 20", event.TokenUsage.InputCacheRead)
	}
	if event.TokenUsage.Output != 30 {
		t.Errorf("Output = %d, want 30", event.TokenUsage.Output)
	}
	if event.SourcePath != filePath {
		t.Errorf("SourcePath = %q, want %q", event.SourcePath, filePath)
	}
	if event.EventID == "" {
		t.Error("EventID should be stable and non-empty")
	}
}

func TestParseCodexUsageEvents_TotalUsageDeltasAcrossDays(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-deltas.jsonl")
	content := `{"timestamp":"2026-04-15T23:50:00Z","type":"session_meta","payload":{"id":"delta-session","timestamp":"2026-04-15T23:50:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T23:51:00Z","type":"turn_context","payload":{"model":"gpt-5.4"}}
{"timestamp":"2026-04-15T23:55:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}}}}
{"timestamp":"2026-04-16T00:10:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":250,"output_tokens":450,"reasoning_output_tokens":0,"total_tokens":1950}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}
	if !events[0].Timestamp.Equal(time.Date(2026, 4, 15, 23, 55, 0, 0, time.UTC)) {
		t.Errorf("first Timestamp = %s, want 2026-04-15T23:55:00Z", events[0].Timestamp.Format(time.RFC3339))
	}
	if events[0].TokenUsage.InputOther != 800 {
		t.Errorf("first InputOther = %d, want 800", events[0].TokenUsage.InputOther)
	}
	if events[0].TokenUsage.InputCacheRead != 200 {
		t.Errorf("first InputCacheRead = %d, want 200", events[0].TokenUsage.InputCacheRead)
	}
	if events[0].TokenUsage.Output != 300 {
		t.Errorf("first Output = %d, want 300", events[0].TokenUsage.Output)
	}
	if !events[1].Timestamp.Equal(time.Date(2026, 4, 16, 0, 10, 0, 0, time.UTC)) {
		t.Errorf("second Timestamp = %s, want 2026-04-16T00:10:00Z", events[1].Timestamp.Format(time.RFC3339))
	}
	if events[1].TokenUsage.InputOther != 450 {
		t.Errorf("second InputOther = %d, want 450", events[1].TokenUsage.InputOther)
	}
	if events[1].TokenUsage.InputCacheRead != 50 {
		t.Errorf("second InputCacheRead = %d, want 50", events[1].TokenUsage.InputCacheRead)
	}
	if events[1].TokenUsage.Output != 150 {
		t.Errorf("second Output = %d, want 150", events[1].TokenUsage.Output)
	}
}

func TestParseCodexUsageEvents_CumulativeResetStartsFreshDelta(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-reset.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"reset-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":100,"output_tokens":200,"total_tokens":1200}}}}
{"timestamp":"2026-04-15T10:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":30,"total_tokens":130}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}
	if events[1].TokenUsage.InputOther != 80 {
		t.Errorf("reset InputOther = %d, want 80", events[1].TokenUsage.InputOther)
	}
	if events[1].TokenUsage.InputCacheRead != 20 {
		t.Errorf("reset InputCacheRead = %d, want 20", events[1].TokenUsage.InputCacheRead)
	}
	if events[1].TokenUsage.Output != 30 {
		t.Errorf("reset Output = %d, want 30", events[1].TokenUsage.Output)
	}
}

func TestParseCodexUsageEvents_LastTokenUsageAdvancesCumulativeBaseline(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-mixed.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"mixed-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
{"timestamp":"2026-04-15T10:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":15,"output_tokens":30,"total_tokens":180}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}
	if events[1].TokenUsage.InputOther != 45 {
		t.Errorf("mixed InputOther = %d, want 45", events[1].TokenUsage.InputOther)
	}
	if events[1].TokenUsage.InputCacheRead != 5 {
		t.Errorf("mixed InputCacheRead = %d, want 5", events[1].TokenUsage.InputCacheRead)
	}
	if events[1].TokenUsage.Output != 10 {
		t.Errorf("mixed Output = %d, want 10", events[1].TokenUsage.Output)
	}
}

func TestParseCodexUsageEvents_LastTokenUsageAfterResetDoesNotDoubleCount(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-reset-after-last.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"reset-last-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":100,"output_tokens":200,"total_tokens":1200}}}}
{"timestamp":"2026-04-15T10:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
{"timestamp":"2026-04-15T10:03:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":15,"output_tokens":30,"total_tokens":180}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3: %#v", len(events), events)
	}
	if events[2].TokenUsage.InputOther != 45 {
		t.Errorf("post-reset InputOther = %d, want 45", events[2].TokenUsage.InputOther)
	}
	if events[2].TokenUsage.InputCacheRead != 5 {
		t.Errorf("post-reset InputCacheRead = %d, want 5", events[2].TokenUsage.InputCacheRead)
	}
	if events[2].TokenUsage.Output != 10 {
		t.Errorf("post-reset Output = %d, want 10", events[2].TokenUsage.Output)
	}

	session, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected session parse error: %v", err)
	}
	if session.TokenUsage.InputOther != 1035 {
		t.Errorf("session InputOther = %d, want 1035", session.TokenUsage.InputOther)
	}
	if session.TokenUsage.InputCacheRead != 115 {
		t.Errorf("session InputCacheRead = %d, want 115", session.TokenUsage.InputCacheRead)
	}
	if session.TokenUsage.Output != 230 {
		t.Errorf("session Output = %d, want 230", session.TokenUsage.Output)
	}
}

func TestParseCodexSession_KeepsFirstSessionMetadataAndSumsResetUsage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-session-reset.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"first-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"first title"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
{"timestamp":"2026-04-15T10:02:00Z","type":"session_meta","payload":{"id":"second-session","timestamp":"2026-04-15T10:02:00Z","cwd":"/other"}}
{"timestamp":"2026-04-15T10:03:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":50,"cached_input_tokens":5,"output_tokens":10,"total_tokens":60}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SessionID != "first-session" {
		t.Errorf("SessionID = %q, want first-session", info.SessionID)
	}
	if info.TokenUsage.InputOther != 135 {
		t.Errorf("InputOther = %d, want 135", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.InputCacheRead != 15 {
		t.Errorf("InputCacheRead = %d, want 15", info.TokenUsage.InputCacheRead)
	}
	if info.TokenUsage.Output != 30 {
		t.Errorf("Output = %d, want 30", info.TokenUsage.Output)
	}
}

func TestParseCodexUsageEvents_KeepsFirstSessionMetadata(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-metadata.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"first-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"first title"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
{"timestamp":"2026-04-15T10:02:00Z","type":"session_meta","payload":{"id":"second-session","timestamp":"2026-04-15T10:02:00Z","cwd":"/other"}}
{"timestamp":"2026-04-15T10:02:01Z","type":"event_msg","payload":{"type":"user_message","message":"second title"}}
{"timestamp":"2026-04-15T10:03:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":20,"output_tokens":30,"total_tokens":180}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}
	for i, event := range events {
		if event.SessionID != "first-session" {
			t.Errorf("event %d SessionID = %q, want first-session", i, event.SessionID)
		}
		if event.Title != "first title" {
			t.Errorf("event %d Title = %q, want first title", i, event.Title)
		}
	}
}

func TestParseCodexUsageEvents_LeavesSessionIDEmptyWithoutMetadata(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-no-metadata.jsonl")
	content := `{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}
	if events[0].SessionID != "" {
		t.Errorf("SessionID = %q, want empty without session_meta.id", events[0].SessionID)
	}
	if events[0].SourcePath != filePath {
		t.Errorf("SourcePath = %q, want %q", events[0].SourcePath, filePath)
	}
}

func TestParseCodexUsageEvents_UsesTurnContextModelFallback(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-model-fallback.jsonl")
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"model-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:00:01Z","type":"turn_context","payload":{"model":"gpt-5.4"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}
	if events[0].ModelName != "gpt-5.4" {
		t.Errorf("ModelName = %q, want gpt-5.4", events[0].ModelName)
	}
}

func TestCollectCodexUsageEvents_UsesCodexHomeWhenBaseDirEmpty(t *testing.T) {
	codexHome := t.TempDir()
	rolloutPath := writeCodexSessionFile(t, filepath.Join(codexHome, "sessions"), "2026", "04", "15", "rollout-home.jsonl", "home-session", "home title")
	t.Setenv("CODEX_HOME", codexHome)

	p := &Provider{}
	events, err := p.CollectUsageEvents("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}
	if events[0].SessionID != "home-session" {
		t.Errorf("SessionID = %q, want home-session", events[0].SessionID)
	}
	if events[0].SourcePath != rolloutPath {
		t.Errorf("SourcePath = %q, want %q", events[0].SourcePath, rolloutPath)
	}
}

func TestCollectCodexSessions_UsesCodexHomeWhenBaseDirEmpty(t *testing.T) {
	codexHome := t.TempDir()
	writeCodexSessionFile(t, filepath.Join(codexHome, "sessions"), "2026", "04", "15", "rollout-home-session.jsonl", "home-session", "home title")
	t.Setenv("CODEX_HOME", codexHome)

	p := &Provider{}
	sessions, err := p.CollectSessions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1: %#v", len(sessions), sessions)
	}
	if sessions[0].SessionID != "home-session" {
		t.Errorf("SessionID = %q, want home-session", sessions[0].SessionID)
	}
}

func TestCollectCodexUsageEvents_ExplicitBaseDirOverridesCodexHome(t *testing.T) {
	codexHome := t.TempDir()
	writeCodexSessionFile(t, filepath.Join(codexHome, "sessions"), "2026", "04", "15", "rollout-home.jsonl", "home-session", "home title")
	explicitDir := t.TempDir()
	writeCodexSessionFile(t, explicitDir, "2026", "04", "16", "rollout-explicit.jsonl", "explicit-session", "explicit title")
	t.Setenv("CODEX_HOME", codexHome)

	p := &Provider{}
	events, err := p.CollectUsageEvents(explicitDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1: %#v", len(events), events)
	}
	if events[0].SessionID != "explicit-session" {
		t.Errorf("SessionID = %q, want explicit-session", events[0].SessionID)
	}
}

func TestCollectCodexUsageEvents_CrossDayDatedLayoutIgnoresFileModTimeForAttribution(t *testing.T) {
	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, "2026", "04", "15")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout-2026-04-15T23-50-00-cross-day.jsonl")
	content := `{"timestamp":"2026-04-15T23:50:00Z","type":"session_meta","payload":{"id":"cross-day-mtime","timestamp":"2026-04-15T23:50:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T23:51:00Z","type":"turn_context","payload":{"model":"gpt-5.4"}}
{"timestamp":"2026-04-15T23:55:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"total_tokens":1300}}}}
{"timestamp":"2026-04-16T00:10:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":250,"output_tokens":450,"total_tokens":1950}}}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	oldModTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, oldModTime, oldModTime); err != nil {
		t.Fatal(err)
	}

	events, err := (&Provider{}).CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %#v", len(events), events)
	}
	if !events[0].Timestamp.Equal(time.Date(2026, 4, 15, 23, 55, 0, 0, time.UTC)) {
		t.Errorf("first Timestamp = %s, want 2026-04-15T23:55:00Z", events[0].Timestamp.Format(time.RFC3339))
	}
	if !events[1].Timestamp.Equal(time.Date(2026, 4, 16, 0, 10, 0, 0, time.UTC)) {
		t.Errorf("second Timestamp = %s, want 2026-04-16T00:10:00Z", events[1].Timestamp.Format(time.RFC3339))
	}
	if events[0].TokenUsage.InputOther != 800 || events[0].TokenUsage.InputCacheRead != 200 || events[0].TokenUsage.Output != 300 {
		t.Errorf("first usage = %+v, want input_other=800 input_cache_read=200 output=300", events[0].TokenUsage)
	}
	if events[1].TokenUsage.InputOther != 450 || events[1].TokenUsage.InputCacheRead != 50 || events[1].TokenUsage.Output != 150 {
		t.Errorf("second usage = %+v, want input_other=450 input_cache_read=50 output=150", events[1].TokenUsage)
	}
	for _, event := range events {
		if event.SessionID != "cross-day-mtime" {
			t.Errorf("SessionID = %q, want cross-day-mtime", event.SessionID)
		}
		if event.SourcePath != path {
			t.Errorf("SourcePath = %q, want %q", event.SourcePath, path)
		}
	}

	filtered := stats.FilterEventsByDateRange(events, "2026-04-16", "2026-04-16", time.UTC)
	if len(filtered) != 1 {
		t.Fatalf("filtered got %d events, want only the in-window event: %#v", len(filtered), filtered)
	}
	if !filtered[0].Timestamp.Equal(time.Date(2026, 4, 16, 0, 10, 0, 0, time.UTC)) {
		t.Errorf("filtered Timestamp = %s, want 2026-04-16T00:10:00Z", filtered[0].Timestamp.Format(time.RFC3339))
	}
	if filtered[0].TokenUsage.InputOther != 450 || filtered[0].TokenUsage.InputCacheRead != 50 || filtered[0].TokenUsage.Output != 150 {
		t.Errorf("filtered usage = %+v, want second event delta only", filtered[0].TokenUsage)
	}
}

func writeCodexSessionFile(t *testing.T, baseDir, year, month, day, name, sessionID, title string) string {
	t.Helper()

	dir := filepath.Join(baseDir, year, month, day)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"` + sessionID + `","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"` + title + `"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":20,"total_tokens":120}}}}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
