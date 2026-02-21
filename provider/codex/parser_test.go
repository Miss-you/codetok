package codex

import (
	"os"
	"path/filepath"
	"testing"
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
