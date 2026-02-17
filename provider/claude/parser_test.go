package claude

import (
	"os"
	"path/filepath"
	"testing"
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
