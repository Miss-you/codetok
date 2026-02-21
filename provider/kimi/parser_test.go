package kimi

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseWireJSONL_ValidData(t *testing.T) {
	usage, turns, startTime, endTime, modelName, err := parseWireJSONL(filepath.Join("testdata", "wire.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 StatusUpdate events: (100+150+200, 50+75+100, 200+300+400, 10+20+30)
	if usage.InputOther != 450 {
		t.Errorf("InputOther = %d, want 450", usage.InputOther)
	}
	if usage.Output != 225 {
		t.Errorf("Output = %d, want 225", usage.Output)
	}
	if usage.InputCacheRead != 900 {
		t.Errorf("InputCacheRead = %d, want 900", usage.InputCacheRead)
	}
	if usage.InputCacheCreate != 60 {
		t.Errorf("InputCacheCreate = %d, want 60", usage.InputCacheCreate)
	}

	if turns != 2 {
		t.Errorf("turns = %d, want 2", turns)
	}

	if usage.TotalInput() != 1410 {
		t.Errorf("TotalInput = %d, want 1410", usage.TotalInput())
	}
	if usage.Total() != 1635 {
		t.Errorf("Total = %d, want 1635", usage.Total())
	}

	if startTime.IsZero() {
		t.Error("startTime should not be zero")
	}
	if endTime.IsZero() {
		t.Error("endTime should not be zero")
	}
	if !endTime.After(startTime) {
		t.Error("endTime should be after startTime")
	}
	if modelName != "" {
		t.Errorf("modelName = %q, want empty", modelName)
	}
}

func TestParseWireJSONL_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	usage, turns, _, _, _, err := parseWireJSONL(emptyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.Total() != 0 {
		t.Errorf("Total = %d, want 0", usage.Total())
	}
	if turns != 0 {
		t.Errorf("turns = %d, want 0", turns)
	}
}

func TestParseWireJSONL_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	content := `{"type": "metadata", "protocol_version": "1.2"}
this is not valid json
{"timestamp": 1770983426.420942, "message": {"type": "StatusUpdate", "payload": {"context_usage": 0.024, "token_usage": {"input_other": 100, "output": 50, "input_cache_read": 200, "input_cache_creation": 10}, "message_id": "chatcmpl-aaa"}}}
`
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, _, _, _, _, err := parseWireJSONL(wirePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have parsed the one valid StatusUpdate
	if usage.InputOther != 100 {
		t.Errorf("InputOther = %d, want 100", usage.InputOther)
	}
	if usage.Output != 50 {
		t.Errorf("Output = %d, want 50", usage.Output)
	}
}

func TestParseWireJSONL_NoStatusUpdate(t *testing.T) {
	dir := t.TempDir()
	content := `{"type": "metadata", "protocol_version": "1.2"}
{"timestamp": 1770983424.646, "message": {"type": "TurnBegin", "payload": {"user_input": []}}}
{"timestamp": 1770983458.818, "message": {"type": "TurnEnd", "payload": {}}}
`
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, turns, _, _, _, err := parseWireJSONL(wirePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.Total() != 0 {
		t.Errorf("Total = %d, want 0", usage.Total())
	}
	if turns != 1 {
		t.Errorf("turns = %d, want 1", turns)
	}
}

func TestParseMetadata_ValidData(t *testing.T) {
	meta, err := parseMetadata(filepath.Join("testdata", "metadata.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.SessionID != "test-session-001" {
		t.Errorf("SessionID = %q, want %q", meta.SessionID, "test-session-001")
	}
	if meta.Title != "Test Session Title" {
		t.Errorf("Title = %q, want %q", meta.Title, "Test Session Title")
	}
	if meta.ModelName != "" {
		t.Errorf("ModelName = %q, want empty", meta.ModelName)
	}
}

func TestParseMetadata_MissingFile(t *testing.T) {
	_, err := parseMetadata(filepath.Join("testdata", "nonexistent.json"))
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestCollectSessions_MultipleSessionDirs(t *testing.T) {
	// Create a temporary directory tree:
	// baseDir/hash1/uuid1/{wire.jsonl, metadata.json}
	// baseDir/hash1/uuid2/{wire.jsonl, metadata.json}
	baseDir := t.TempDir()

	sessions := []struct {
		hash      string
		uuid      string
		sessionID string
		title     string
	}{
		{"hashA", "uuid-1", "session-1", "First Session"},
		{"hashA", "uuid-2", "session-2", "Second Session"},
		{"hashB", "uuid-3", "session-3", "Third Session"},
	}

	wireContent := `{"type": "metadata", "protocol_version": "1.2"}
{"timestamp": 1770983424.646, "message": {"type": "TurnBegin", "payload": {"user_input": [{"type": "text", "text": "hi"}]}}}
{"timestamp": 1770983426.420, "message": {"type": "StatusUpdate", "payload": {"context_usage": 0.024, "token_usage": {"input_other": 100, "output": 50, "input_cache_read": 200, "input_cache_creation": 0}, "message_id": "msg-1"}}}
{"timestamp": 1770983458.818, "message": {"type": "TurnEnd", "payload": {}}}
`

	for _, s := range sessions {
		dir := filepath.Join(baseDir, s.hash, s.uuid)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "wire.jsonl"), []byte(wireContent), 0644); err != nil {
			t.Fatal(err)
		}
		metaJSON := `{"session_id": "` + s.sessionID + `", "title": "` + s.title + `"}`
		if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(metaJSON), 0644); err != nil {
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
		if s.TokenUsage.InputOther != 100 {
			t.Errorf("session %s InputOther = %d, want 100", s.SessionID, s.TokenUsage.InputOther)
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

func TestTimestampExtraction(t *testing.T) {
	usage, _, startTime, endTime, _, err := parseWireJSONL(filepath.Join("testdata", "wire.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = usage

	// First TurnBegin timestamp: 1770983424.646556
	expectedStart := time.Unix(1770983424, 646556000)
	// Last TurnEnd timestamp: 1770983779.790828
	expectedEnd := time.Unix(1770983779, 790828000)

	// Allow small tolerance for float precision
	if startTime.Sub(expectedStart).Abs() > time.Millisecond {
		t.Errorf("startTime = %v, want close to %v", startTime, expectedStart)
	}
	if endTime.Sub(expectedEnd).Abs() > time.Millisecond {
		t.Errorf("endTime = %v, want close to %v", endTime, expectedEnd)
	}
}

func TestParseWireJSONL_ModelExtraction(t *testing.T) {
	dir := t.TempDir()
	content := `{"type": "metadata", "protocol_version": "1.2"}
{"timestamp": 1770983424.646, "message": {"type": "TurnBegin", "payload": {"user_input": []}}}
{"timestamp": 1770983426.420, "message": {"type": "StatusUpdate", "payload": {"model_name":"moonshot-v1-128k","token_usage": {"input_other": 100, "output": 50, "input_cache_read": 200, "input_cache_creation": 10}}}}
{"timestamp": 1770983458.818, "message": {"type": "TurnEnd", "payload": {}}}
`
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, _, _, modelName, err := parseWireJSONL(wirePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modelName != "moonshot-v1-128k" {
		t.Errorf("modelName = %q, want %q", modelName, "moonshot-v1-128k")
	}
}

func TestParseSession_MetadataModelFallback(t *testing.T) {
	baseDir := t.TempDir()
	sessionDir := filepath.Join(baseDir, "hashA", "uuid-1")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	wireContent := `{"type": "metadata", "protocol_version": "1.2"}
{"timestamp": 1770983424.646, "message": {"type": "TurnBegin", "payload": {"user_input": []}}}
{"timestamp": 1770983458.818, "message": {"type": "TurnEnd", "payload": {}}}
`
	if err := os.WriteFile(filepath.Join(sessionDir, "wire.jsonl"), []byte(wireContent), 0644); err != nil {
		t.Fatal(err)
	}

	metaJSON := `{"session_id":"session-1","title":"Session","model_name":"moonshot-v1-32k"}`
	if err := os.WriteFile(filepath.Join(sessionDir, "metadata.json"), []byte(metaJSON), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionDir, "hashA", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ModelName != "moonshot-v1-32k" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "moonshot-v1-32k")
	}
}

func TestNormalizeKimiModelName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "k2.5 short alias", in: "k2.5", want: "kimi-k2.5"},
		{name: "k2.5 mixed case", in: " K2.5 ", want: "kimi-k2.5"},
		{name: "k2-thinking short alias", in: "k2_thinking", want: "kimi-k2-thinking"},
		{name: "k2-thinking canonical", in: "KIMI-K2-THINKING", want: "kimi-k2-thinking"},
		{name: "unknown model unchanged", in: "moonshot-v1-32k", want: "moonshot-v1-32k"},
		{name: "empty model", in: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeKimiModelName(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeKimiModelName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestModelNameFromLogLine_Variants(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "single quote fields",
			line: "2026-02-20T10:00:01Z INFO Using LLM model: provider='moonshot' model='K2.5'",
			want: "K2.5",
		},
		{
			name: "double quote fields with spacing",
			line: "2026-02-20T10:00:01Z INFO Using LLM model : provider = \"moonshot\"   model_name = \"kimi-k2-thinking\"",
			want: "kimi-k2-thinking",
		},
		{
			name: "upper case marker",
			line: "2026-02-20T10:00:01Z INFO USING LLM MODEL: provider='moonshot' model_id='k2.5'",
			want: "k2.5",
		},
		{
			name: "non marker line",
			line: "2026-02-20T10:00:01Z INFO model='k2.5'",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelNameFromLogLine(tt.line)
			if got != tt.want {
				t.Fatalf("modelNameFromLogLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestMergeSessionModelsFromLog_UsesLatestModelForSession(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "kimi-main.log")
	sessionID := "11111111-2222-3333-4444-555555555555"
	content := `2026-02-20T10:00:00Z INFO Created new session: ` + sessionID + `
2026-02-20T10:00:01Z INFO Using LLM model: provider='moonshot' model='K2.5'
2026-02-20T10:00:02Z INFO Using LLM model: provider='moonshot' model='k2-thinking'
`
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	sessionModelIndex := make(map[string]string)
	mergeSessionModelsFromLog(logPath, sessionModelIndex)

	got, exists := sessionModelIndex[normalizeSessionIDForLookup(sessionID)]
	if !exists {
		t.Fatalf("missing sessionID mapping for %q", sessionID)
	}
	if got != "kimi-k2-thinking" {
		t.Fatalf("model mapping = %q, want %q", got, "kimi-k2-thinking")
	}
}

func TestCollectSessions_ModelFallbackFromLogs(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	logsDir := filepath.Join(homeDir, ".kimi", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionID := "11111111-2222-3333-4444-555555555555"
	logContent := `2026-02-20T10:00:00Z INFO Created new session: ` + sessionID + `
2026-02-20T10:00:01Z INFO Using LLM model: provider='moonshot' model='K2.5'
`
	if err := os.WriteFile(filepath.Join(logsDir, "kimi-main.log"), []byte(logContent), 0644); err != nil {
		t.Fatal(err)
	}

	baseDir := filepath.Join(homeDir, ".kimi", "sessions")
	sessionDir := filepath.Join(baseDir, "hashA", sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}

	wireContent := `{"timestamp": 1770983424.646, "message": {"type": "TurnBegin", "payload": {"user_input": []}}}
{"timestamp": 1770983426.420, "message": {"type": "StatusUpdate", "payload": {"token_usage": {"input_other": 100, "output": 50, "input_cache_read": 200, "input_cache_creation": 10}}}}
{"timestamp": 1770983458.818, "message": {"type": "TurnEnd", "payload": {}}}
`
	if err := os.WriteFile(filepath.Join(sessionDir, "wire.jsonl"), []byte(wireContent), 0644); err != nil {
		t.Fatal(err)
	}

	metaContent := `{"session_id":"` + sessionID + `","title":"Session Without Model"}`
	if err := os.WriteFile(filepath.Join(sessionDir, "metadata.json"), []byte(metaContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	sessions, err := p.CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("CollectSessions returned error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	if sessions[0].ModelName != "kimi-k2.5" {
		t.Fatalf("ModelName = %q, want %q", sessions[0].ModelName, "kimi-k2.5")
	}
}
