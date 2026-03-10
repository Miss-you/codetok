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

func TestCollectClaudeSessions_MissingDirectoryGraceful(t *testing.T) {
	// Point to a non-existent directory — collectPaths should skip it
	baseDir := filepath.Join(t.TempDir(), "nonexistent")

	prov := &Provider{}
	result, err := prov.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d sessions, want 0", len(result))
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
	collectPaths(dir1, &paths, pathToSlug)
	collectPaths(dir2, &paths, pathToSlug)

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
