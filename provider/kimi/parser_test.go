package kimi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

func TestParseWireJSONL_ValidData(t *testing.T) {
	usage, turns, startTime, endTime, modelName, err := parseWireJSONL(filepath.Join("testdata", "wire.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 usage.record events: (100+150+200, 50+75+100, 200+300+400, 10+20+30)
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
	if modelName != "kimi-code/k3-256k" {
		t.Errorf("modelName = %q, want %q", modelName, "kimi-code/k3-256k")
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
	content := `{"type":"metadata","protocol_version":"1.5","created_at":1770983424646}
this is not valid json
{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":100,"output":50,"inputCacheRead":200,"inputCacheCreation":10},"usageScope":"turn","time":1770983426420}
`
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, _, _, _, _, err := parseWireJSONL(wirePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have parsed the one valid usage.record
	if usage.InputOther != 100 {
		t.Errorf("InputOther = %d, want 100", usage.InputOther)
	}
	if usage.Output != 50 {
		t.Errorf("Output = %d, want 50", usage.Output)
	}
}

func TestParseWireJSONL_NoUsageRecord(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"metadata","protocol_version":"1.5","created_at":1770983424646}
{"type":"turn.prompt","input":[{"type":"text","text":"hi"}],"origin":{"kind":"user"},"time":1770983424646}
{"type":"turn.ended","turnId":0,"reason":"completed","durationMs":4000,"time":1770983458818}
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

func TestParseSessionState_ValidData(t *testing.T) {
	state, err := parseSessionState(filepath.Join("testdata", "state.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Title != "Test Session Title" {
		t.Errorf("Title = %q, want %q", state.Title, "Test Session Title")
	}
}

func TestParseSessionState_MissingFile(t *testing.T) {
	_, err := parseSessionState(filepath.Join("testdata", "nonexistent.json"))
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestCollectSessions_MultipleSessionDirs(t *testing.T) {
	// Create a temporary directory tree:
	// baseDir/<work-dir>/<session-id>/{state.json, agents/main/wire.jsonl}
	baseDir := t.TempDir()

	sessions := []struct {
		workDir   string
		sessionID string
		title     string
	}{
		{"wd_a", "session-1", "First Session"},
		{"wd_a", "session-2", "Second Session"},
		{"wd_b", "session-3", "Third Session"},
	}

	wireContent := `{"type":"metadata","protocol_version":"1.5","created_at":1770983424646}
{"type":"turn.prompt","input":[{"type":"text","text":"hi"}],"origin":{"kind":"user"},"time":1770983424646}
{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":100,"output":50,"inputCacheRead":200,"inputCacheCreation":0},"usageScope":"turn","time":1770983426420}
{"type":"turn.ended","turnId":0,"reason":"completed","durationMs":4000,"time":1770983458818}
`

	for _, s := range sessions {
		dir := filepath.Join(baseDir, s.workDir, s.sessionID, "agents", "main")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "wire.jsonl"), []byte(wireContent), 0644); err != nil {
			t.Fatal(err)
		}
		stateJSON := `{"title": "` + s.title + `"}`
		if err := os.WriteFile(filepath.Join(baseDir, s.workDir, s.sessionID, "state.json"), []byte(stateJSON), 0644); err != nil {
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
		if s.ModelName != "k3-256k" {
			t.Errorf("session %s ModelName = %q, want %q", s.SessionID, s.ModelName, "k3-256k")
		}
	}
	for _, s := range sessions {
		if !ids[s.sessionID] {
			t.Errorf("missing session %s", s.sessionID)
		}
	}
}

func TestParseSession_MultiAgentAggregation(t *testing.T) {
	baseDir := t.TempDir()
	sessionDir := filepath.Join(baseDir, "wd_a", "session-1")

	mainWire := `{"type":"metadata","protocol_version":"1.5","created_at":1770983424646}
{"type":"turn.prompt","input":[{"type":"text","text":"hi"}],"origin":{"kind":"user"},"time":1770983424646}
{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":100,"output":50,"inputCacheRead":200,"inputCacheCreation":10},"usageScope":"turn","time":1770983426420}
{"type":"turn.ended","turnId":0,"reason":"completed","durationMs":4000,"time":1770983458818}
`
	subWire := `{"type":"metadata","protocol_version":"1.5","created_at":1770983425000}
{"type":"turn.prompt","input":[{"type":"text","text":"sub task"}],"origin":{"kind":"user"},"time":1770983425000}
{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":40,"output":10,"inputCacheRead":0,"inputCacheCreation":0},"usageScope":"turn","time":1770983430000}
`
	for agent, content := range map[string]string{"main": mainWire, "agent-0": subWire} {
		dir := filepath.Join(sessionDir, "agents", agent)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "wire.jsonl"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "state.json"), []byte(`{"title":"Multi Agent"}`), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := parseSession(sessionDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.TokenUsage.InputOther != 140 {
		t.Errorf("InputOther = %d, want 140 (main+subagent)", info.TokenUsage.InputOther)
	}
	if info.TokenUsage.Output != 60 {
		t.Errorf("Output = %d, want 60 (main+subagent)", info.TokenUsage.Output)
	}
	// Only the main agent's prompts count as turns.
	if info.Turns != 1 {
		t.Errorf("Turns = %d, want 1", info.Turns)
	}
	if info.Title != "Multi Agent" {
		t.Errorf("Title = %q, want %q", info.Title, "Multi Agent")
	}
	if info.ModelName != "k3-256k" {
		t.Errorf("ModelName = %q, want %q", info.ModelName, "k3-256k")
	}
	if info.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want %q", info.SessionID, "session-1")
	}
}

func TestTimestampExtraction(t *testing.T) {
	_, _, startTime, endTime, _, err := parseWireJSONL(filepath.Join("testdata", "wire.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Earliest event time: 1770983424646 ms (first turn.prompt)
	expectedStart := time.UnixMilli(1770983424646)
	// Latest event time: 1770983779790 ms (last turn.ended)
	expectedEnd := time.UnixMilli(1770983779790)

	if !startTime.Equal(expectedStart) {
		t.Errorf("startTime = %v, want %v", startTime, expectedStart)
	}
	if !endTime.Equal(expectedEnd) {
		t.Errorf("endTime = %v, want %v", endTime, expectedEnd)
	}
}

func TestParseKimiUsageEvents_UsageRecordsEmitIncrementalEvents(t *testing.T) {
	dir := t.TempDir()
	firstTimestamp := time.Date(2026, 2, 15, 23, 59, 59, 500000000, time.UTC)
	secondTimestamp := firstTimestamp.Add(time.Second)
	content := fmt.Sprintf(`{"type":"metadata","protocol_version":"1.5","created_at":%d}
{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":100,"output":50,"inputCacheRead":200,"inputCacheCreation":10},"usageScope":"turn","time":%d}
{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":150,"output":75,"inputCacheRead":300,"inputCacheCreation":20},"usageScope":"turn","time":%d}
{"type":"usage.record","usage":{"inputOther":1,"output":2,"inputCacheRead":3,"inputCacheCreation":4},"usageScope":"turn","time":%d}
`, firstTimestamp.UnixMilli(), firstTimestamp.UnixMilli(), secondTimestamp.UnixMilli(), secondTimestamp.Add(time.Second).UnixMilli())
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	baseEvent := provider.UsageEvent{
		ProviderName: "kimi",
		SessionID:    "session-1",
		Title:        "Session",
		WorkDirHash:  "wd_a",
	}
	events, modelName, err := parseKimiUsageEvents(wirePath, baseEvent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if modelName != "kimi-code/k3-256k" {
		t.Fatalf("modelName = %q, want %q", modelName, "kimi-code/k3-256k")
	}

	if got, want := events[0].Timestamp, firstTimestamp; !got.Equal(want) {
		t.Fatalf("first Timestamp = %v, want %v", got, want)
	}
	if got, want := events[1].Timestamp, secondTimestamp; !got.Equal(want) {
		t.Fatalf("second Timestamp = %v, want %v", got, want)
	}
	if events[0].Timestamp.In(time.UTC).Day() == events[1].Timestamp.In(time.UTC).Day() {
		t.Fatalf("events should remain separate across days: %v and %v", events[0].Timestamp, events[1].Timestamp)
	}

	if events[0].TokenUsage.InputOther != 100 || events[0].TokenUsage.Output != 50 || events[0].TokenUsage.InputCacheRead != 200 || events[0].TokenUsage.InputCacheCreate != 10 {
		t.Fatalf("first TokenUsage = %+v, want line-local usage", events[0].TokenUsage)
	}
	if events[1].TokenUsage.InputOther != 150 || events[1].TokenUsage.Output != 75 || events[1].TokenUsage.InputCacheRead != 300 || events[1].TokenUsage.InputCacheCreate != 20 {
		t.Fatalf("second TokenUsage = %+v, want line-local usage", events[1].TokenUsage)
	}
	if events[2].TokenUsage.InputOther != 1 || events[2].TokenUsage.Output != 2 || events[2].TokenUsage.InputCacheRead != 3 || events[2].TokenUsage.InputCacheCreate != 4 {
		t.Fatalf("third TokenUsage = %+v, want line-local usage", events[2].TokenUsage)
	}
	if events[0].EventID != wirePath+":2" {
		t.Fatalf("first EventID = %q, want %q", events[0].EventID, wirePath+":2")
	}
	if events[0].ModelName != "k3-256k" {
		t.Fatalf("first ModelName = %q, want %q", events[0].ModelName, "k3-256k")
	}
	// The record without a model falls back to the file-level model.
	if events[2].ModelName != "" {
		t.Fatalf("third ModelName = %q, want empty (fallback happens in parseSessionUsageEvents)", events[2].ModelName)
	}
	for i, event := range events {
		if event.ProviderName != "kimi" || event.SessionID != "session-1" || event.Title != "Session" || event.WorkDirHash != "wd_a" {
			t.Fatalf("event %d metadata = %+v, want base metadata preserved", i, event)
		}
		if event.SourcePath != wirePath {
			t.Fatalf("event %d SourcePath = %q, want %q", i, event.SourcePath, wirePath)
		}
	}
}

func TestCollectKimiUsageEvents_MultiAgentSessions(t *testing.T) {
	baseDir := t.TempDir()
	sessionDir := filepath.Join(baseDir, "wd_a", "session-1")

	for agent, input := range map[string]int{"main": 100, "agent-0": 40} {
		dir := filepath.Join(sessionDir, "agents", agent)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		content := fmt.Sprintf(`{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":%d,"output":10,"inputCacheRead":0,"inputCacheCreation":0},"usageScope":"turn","time":1776297300000}`+"\n", input)
		if err := os.WriteFile(filepath.Join(dir, "wire.jsonl"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "state.json"), []byte(`{"title":"Multi Agent"}`), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	events, err := p.CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("CollectUsageEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	totalInput := 0
	for _, event := range events {
		if event.SessionID != "session-1" {
			t.Fatalf("SessionID = %q, want %q", event.SessionID, "session-1")
		}
		if event.Title != "Multi Agent" {
			t.Fatalf("Title = %q, want %q", event.Title, "Multi Agent")
		}
		if event.ModelName != "k3-256k" {
			t.Fatalf("ModelName = %q, want %q", event.ModelName, "k3-256k")
		}
		if event.WorkDirHash != "wd_a" {
			t.Fatalf("WorkDirHash = %q, want %q", event.WorkDirHash, "wd_a")
		}
		totalInput += event.TokenUsage.InputOther
	}
	if totalInput != 140 {
		t.Fatalf("total InputOther = %d, want 140 (main+subagent)", totalInput)
	}
}

func TestCollectKimiUsageEvents_RecordWithoutModelUsesFileModel(t *testing.T) {
	baseDir := t.TempDir()
	sessionDir := filepath.Join(baseDir, "wd_a", "session-1")
	wireDir := filepath.Join(sessionDir, "agents", "main")
	if err := os.MkdirAll(wireDir, 0755); err != nil {
		t.Fatal(err)
	}
	wireContent := `{"type":"usage.record","model":"kimi-code/kimi-for-coding","usage":{"inputOther":100,"output":50,"inputCacheRead":200,"inputCacheCreation":10},"usageScope":"turn","time":1776297300000}
{"type":"usage.record","usage":{"inputOther":1,"output":2,"inputCacheRead":3,"inputCacheCreation":4},"usageScope":"session","time":1776297900000}
`
	if err := os.WriteFile(filepath.Join(wireDir, "wire.jsonl"), []byte(wireContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Provider{}
	events, err := p.CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("CollectUsageEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	for i, event := range events {
		if event.ModelName != "kimi-for-coding" {
			t.Fatalf("event %d ModelName = %q, want %q", i, event.ModelName, "kimi-for-coding")
		}
	}
}

func TestCollectKimiUsageEventsInRange_SkipsInactiveWireFilesByModTime(t *testing.T) {
	baseDir := t.TempDir()
	oldWire := writeKimiUsageSession(t, baseDir, "work-a", "old-session", time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), 100)
	activeWire := writeKimiUsageSession(t, baseDir, "work-a", "active-session", time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC), 200)
	oldModTime := time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)
	activeModTime := time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldWire, oldModTime, oldModTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(activeWire, activeModTime, activeModTime); err != nil {
		t.Fatal(err)
	}

	var metrics provider.UsageEventCollectMetrics
	events, err := (&Provider{}).CollectUsageEventsInRange(baseDir, provider.UsageEventCollectOptions{
		Since:    time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		Location: time.UTC,
		Metrics:  &metrics,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].SessionID != "active-session" {
		t.Fatalf("events = %#v, want only active-session", events)
	}
	if metrics.ConsideredFiles != 2 || metrics.SkippedFiles != 1 || metrics.ParsedFiles != 1 || metrics.EmittedEvents != 1 {
		t.Fatalf("metrics = %+v, want considered=2 skipped=1 parsed=1 emitted=1", metrics)
	}
}

func TestCollectKimiUsageEventsInRange_KeepsSessionModifiedAfterUntil(t *testing.T) {
	baseDir := t.TempDir()
	wirePath := writeKimiUsageSession(t, baseDir, "work-a", "cross-day-session", time.Date(2026, 4, 16, 23, 30, 0, 0, time.UTC), 300)
	afterUntil := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(wirePath, afterUntil, afterUntil); err != nil {
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
	if len(events) != 1 || events[0].SessionID != "cross-day-session" {
		t.Fatalf("events = %#v, want cross-day-session candidate", events)
	}
	if metrics.ConsideredFiles != 1 || metrics.SkippedFiles != 0 || metrics.ParsedFiles != 1 || metrics.EmittedEvents != 1 {
		t.Fatalf("metrics = %+v, want considered=1 skipped=0 parsed=1 emitted=1", metrics)
	}
}

func writeKimiUsageSession(t *testing.T, baseDir, workDirName, sessionID string, timestamp time.Time, input int) string {
	t.Helper()
	sessionDir := filepath.Join(baseDir, workDirName, sessionID)
	wireDir := filepath.Join(sessionDir, "agents", "main")
	if err := os.MkdirAll(wireDir, 0755); err != nil {
		t.Fatal(err)
	}
	state := fmt.Sprintf(`{"title":"%s"}`, sessionID)
	if err := os.WriteFile(filepath.Join(sessionDir, "state.json"), []byte(state), 0644); err != nil {
		t.Fatal(err)
	}
	wirePath := filepath.Join(wireDir, "wire.jsonl")
	content := fmt.Sprintf(`{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":%d,"output":10,"inputCacheRead":0,"inputCacheCreation":0},"usageScope":"turn","time":%d}`+"\n",
		input,
		timestamp.UnixMilli(),
	)
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return wirePath
}

func TestNormalizeKimiModelName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "kimi code profile alias", in: "kimi-code/k3-256k", want: "k3-256k"},
		{name: "kimi code default model", in: "kimi-code/kimi-for-coding", want: "kimi-for-coding"},
		{name: "whitespace trimmed", in: " kimi-code/k3-256k ", want: "k3-256k"},
		{name: "other namespace unchanged", in: "openai/gpt-5", want: "openai/gpt-5"},
		{name: "plain model unchanged", in: "k3-256k", want: "k3-256k"},
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

func TestParseWireJSONL_LongLines(t *testing.T) {
	// profile.bind embeds the full system prompt and can exceed 1MB; the parser
	// must not stop at such lines.
	dir := t.TempDir()
	longPrompt := strings.Repeat("x", 2*1024*1024)
	content := `{"type":"profile.bind","modelAlias":"kimi-code/k3-256k","profileName":"agent","systemPrompt":"` + longPrompt + `","time":1770983424650}` + "\n" +
		`{"type":"usage.record","model":"kimi-code/k3-256k","usage":{"inputOther":100,"output":50,"inputCacheRead":200,"inputCacheCreation":10},"usageScope":"turn","time":1770983426420}` + "\n"
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	usage, _, _, _, modelName, err := parseWireJSONL(wirePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.InputOther != 100 {
		t.Errorf("InputOther = %d, want 100", usage.InputOther)
	}
	if modelName != "kimi-code/k3-256k" {
		t.Errorf("modelName = %q, want %q", modelName, "kimi-code/k3-256k")
	}
}

func TestParseWireJSONL_TurnPromptOriginFilter(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"turn.prompt","input":[{"type":"text","text":"hi"}],"origin":{"kind":"user"},"time":1770983424646}
{"type":"turn.prompt","input":[],"origin":{"kind":"system_trigger","name":"goal_continuation"},"time":1770983430000}
{"type":"turn.prompt","input":[],"origin":{"kind":"retry"},"time":1770983435000}
{"type":"turn.prompt","input":[{"type":"text","text":"legacy"}],"time":1770983440000}
{"type":"turn.ended","turnId":0,"reason":"completed","time":1770983458818}
`
	wirePath := filepath.Join(dir, "wire.jsonl")
	if err := os.WriteFile(wirePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, turns, _, _, _, err := parseWireJSONL(wirePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// user prompt + legacy prompt without origin count; system_trigger and retry do not.
	if turns != 2 {
		t.Errorf("turns = %d, want 2", turns)
	}
}
