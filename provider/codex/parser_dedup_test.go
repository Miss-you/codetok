package codex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

// writeCodexRollout writes a rollout fixture in the dated layout:
// baseDir/<year>/<month>/<day>/<name>.
func writeCodexRollout(t *testing.T, baseDir, day, name, content string) string {
	t.Helper()
	dir := filepath.Join(baseDir, "2026", "04", day)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// forkReplayFixtures builds a root session plus two forks that replay its
// cumulative token stream, mirroring Codex Desktop subagent thread files.
//
// Root (04-15): totals t1=(1000,200,300,0,1300) then t2=(1500,250,450,0,1950)
// Fork B (04-16): replays t1, t2, then its own t3=(2000,400,600,0,2600)
// Fork C (04-16): replays t1, t2, t3 with no tokens of its own
func forkReplayFixtures(t *testing.T) string {
	t.Helper()
	baseDir := t.TempDir()

	writeCodexRollout(t, baseDir, "15", "rollout-root.jsonl", `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"rollout-root","session_id":"thread-1","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}}}}
{"timestamp":"2026-04-15T10:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":250,"output_tokens":450,"reasoning_output_tokens":0,"total_tokens":1950}}}}
`)
	writeCodexRollout(t, baseDir, "16", "rollout-fork-b.jsonl", `{"timestamp":"2026-04-16T09:00:00Z","type":"session_meta","payload":{"id":"rollout-fork-b","session_id":"thread-1","forked_from_id":"rollout-root","timestamp":"2026-04-16T09:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-16T09:00:00.100Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}}}}
{"timestamp":"2026-04-16T09:00:00.200Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":250,"output_tokens":450,"reasoning_output_tokens":0,"total_tokens":1950}}}}
{"timestamp":"2026-04-16T09:05:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":2000,"cached_input_tokens":400,"output_tokens":600,"reasoning_output_tokens":0,"total_tokens":2600}}}}
`)
	writeCodexRollout(t, baseDir, "16", "rollout-fork-c.jsonl", `{"timestamp":"2026-04-16T09:30:00Z","type":"session_meta","payload":{"id":"rollout-fork-c","session_id":"thread-1","forked_from_id":"rollout-root","timestamp":"2026-04-16T09:30:00Z","cwd":"/test"}}
{"timestamp":"2026-04-16T09:30:00.100Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}}}}
{"timestamp":"2026-04-16T09:30:00.200Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":250,"output_tokens":450,"reasoning_output_tokens":0,"total_tokens":1950}}}}
{"timestamp":"2026-04-16T09:30:00.300Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":2000,"cached_input_tokens":400,"output_tokens":600,"reasoning_output_tokens":0,"total_tokens":2600}}}}
`)
	return baseDir
}

func TestCollectCodexUsageEvents_ForkReplayDeduplicated(t *testing.T) {
	baseDir := forkReplayFixtures(t)

	events, err := (&Provider{}).CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (replays deduplicated): %#v", len(events), events)
	}

	var totalInputOther, totalCacheRead, totalOutput int
	seen := make(map[string]bool)
	for _, e := range events {
		totalInputOther += e.TokenUsage.InputOther
		totalCacheRead += e.TokenUsage.InputCacheRead
		totalOutput += e.TokenUsage.Output
		seen[e.Timestamp.Format(time.RFC3339Nano)] = true
	}
	// True usage is the root stream plus fork B's own increment:
	// input_other: 800 + 450 + 350, cache_read: 200 + 50 + 150, output: 300 + 150 + 150
	if totalInputOther != 1600 {
		t.Errorf("total InputOther = %d, want 1600", totalInputOther)
	}
	if totalCacheRead != 400 {
		t.Errorf("total InputCacheRead = %d, want 400", totalCacheRead)
	}
	if totalOutput != 600 {
		t.Errorf("total Output = %d, want 600", totalOutput)
	}
	// Replayed increments keep the original (earliest) timestamps.
	for _, want := range []string{
		"2026-04-15T10:01:00Z",
		"2026-04-15T10:02:00Z",
		"2026-04-16T09:05:00Z",
	} {
		ts, err := time.Parse(time.RFC3339, want)
		if err != nil {
			t.Fatal(err)
		}
		if !seen[ts.Format(time.RFC3339Nano)] {
			t.Errorf("missing event at %s", want)
		}
	}
}

func TestCollectCodexUsageEvents_DistinctThreadsNotMerged(t *testing.T) {
	baseDir := t.TempDir()
	tuple := `"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}`
	writeCodexRollout(t, baseDir, "15", "rollout-x.jsonl", fmt.Sprintf(`{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"rollout-x","session_id":"thread-1","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{%s}}}
`, tuple))
	writeCodexRollout(t, baseDir, "15", "rollout-y.jsonl", fmt.Sprintf(`{"timestamp":"2026-04-15T11:00:00Z","type":"session_meta","payload":{"id":"rollout-y","session_id":"thread-2","timestamp":"2026-04-15T11:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T11:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{%s}}}
`, tuple))

	events, err := (&Provider{}).CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (distinct threads must not merge)", len(events))
	}
}

func TestCollectCodexUsageEvents_OldFormatWithoutSessionIDNotMerged(t *testing.T) {
	baseDir := t.TempDir()
	tuple := `"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}`
	writeCodexRollout(t, baseDir, "15", "rollout-old1.jsonl", fmt.Sprintf(`{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"rollout-old1","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{%s}}}
`, tuple))
	writeCodexRollout(t, baseDir, "15", "rollout-old2.jsonl", fmt.Sprintf(`{"timestamp":"2026-04-15T11:00:00Z","type":"session_meta","payload":{"id":"rollout-old2","timestamp":"2026-04-15T11:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T11:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{%s}}}
`, tuple))

	events, err := (&Provider{}).CollectUsageEvents(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (old-format files keep per-file identity)", len(events))
	}
}

func TestCollectCodexSessions_ForkReplayNetUsage(t *testing.T) {
	baseDir := forkReplayFixtures(t)

	sessions, err := (&Provider{}).CollectSessions(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byID := make(map[string]int)
	for i, s := range sessions {
		byID[s.SessionID] = i
	}
	sessionByID := func(id string) provider.SessionInfo {
		i, ok := byID[id]
		if !ok {
			t.Fatalf("session %q not found in %#v", id, sessions)
		}
		return sessions[i]
	}

	root := sessionByID("rollout-root")
	if root.TokenUsage.InputOther != 1250 || root.TokenUsage.InputCacheRead != 250 || root.TokenUsage.Output != 450 {
		t.Errorf("root usage = %+v, want {InputOther:1250 Output:450 CacheRead:250}", root.TokenUsage)
	}
	forkB := sessionByID("rollout-fork-b")
	if forkB.TokenUsage.InputOther != 350 || forkB.TokenUsage.InputCacheRead != 150 || forkB.TokenUsage.Output != 150 {
		t.Errorf("fork B usage = %+v, want only its own increment {InputOther:350 Output:150 CacheRead:150}", forkB.TokenUsage)
	}
	forkC := sessionByID("rollout-fork-c")
	if forkC.TokenUsage.Total() != 0 {
		t.Errorf("fork C usage = %+v, want zero (pure replay)", forkC.TokenUsage)
	}
}

func TestParseCodexUsageEvents_LongLines(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "rollout-long.jsonl")
	longMessage := strings.Repeat("x", 3*1024*1024)
	content := `{"timestamp":"2026-04-15T10:00:00Z","type":"session_meta","payload":{"id":"long-session","timestamp":"2026-04-15T10:00:00Z","cwd":"/test"}}
{"timestamp":"2026-04-15T10:00:02Z","type":"event_msg","payload":{"type":"user_message","message":"` + longMessage + `"}}
{"timestamp":"2026-04-15T10:01:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":0,"total_tokens":1300}}}}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	events, err := parseCodexUsageEvents(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (file with >1MB line must not be dropped)", len(events))
	}

	info, err := parseCodexSession(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenUsage.Total() != 1300 {
		t.Errorf("session usage Total = %d, want 1300", info.TokenUsage.Total())
	}
}
