package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	cursorapi "github.com/miss-you/codetok/cursor"
	"github.com/miss-you/codetok/provider"
)

// testdataDir returns the absolute path to the e2e testdata/sessions directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "sessions"))
	if err != nil {
		t.Fatalf("failed to get testdata dir: %v", err)
	}
	return dir
}

// claudeTestdataDir returns the absolute path to the e2e testdata/claude-sessions directory.
func claudeTestdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "claude-sessions"))
	if err != nil {
		t.Fatalf("failed to get claude testdata dir: %v", err)
	}
	return dir
}

// cursorTestdataDir returns the absolute path to the e2e testdata/cursor directory.
func cursorTestdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "cursor"))
	if err != nil {
		t.Fatalf("failed to get cursor testdata dir: %v", err)
	}
	return dir
}

// emptyDir returns a path to an empty temp directory (to isolate providers in tests).
func emptyDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// isolatedArgs returns args that point all providers to empty dirs except the one being tested.
func isolatedArgs(t *testing.T, extraArgs ...string) []string {
	t.Helper()
	empty := emptyDir(t)
	base := []string{"--claude-dir", empty, "--codex-dir", empty, "--cursor-dir", empty}
	return append(base, extraArgs...)
}

// buildBinary builds the codetok binary and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "codetok")

	moduleRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("failed to get module root: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", binPath, moduleRoot)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build codetok: %v\n%s", err, out)
	}
	return binPath
}

// runCodetok runs the codetok binary with the given arguments and returns stdout.
func runCodetok(t *testing.T, binPath string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %v failed: %v\nstderr: %s", args, err, stderr.String())
	}
	return stdout.String()
}

type activityFixtureRow struct {
	composerAdded   int
	composerDeleted int
	tabAdded        int
	tabDeleted      int
}

func writeCursorActivityDB(t *testing.T, rows []activityFixtureRow) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ai-code-tracking.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE scored_commits (
	commitHash TEXT NOT NULL,
	branchName TEXT NOT NULL,
	scoredAt INTEGER NOT NULL,
	linesAdded INTEGER,
	linesDeleted INTEGER,
	tabLinesAdded INTEGER,
	tabLinesDeleted INTEGER,
	composerLinesAdded INTEGER,
	composerLinesDeleted INTEGER,
	humanLinesAdded INTEGER,
	humanLinesDeleted INTEGER,
	blankLinesAdded INTEGER,
	blankLinesDeleted INTEGER,
	commitMessage TEXT,
	commitDate TEXT,
	v1AiPercentage TEXT,
	v2AiPercentage TEXT,
	PRIMARY KEY (commitHash, branchName)
);
`)
	if err != nil {
		t.Fatalf("creating scored_commits table: %v", err)
	}

	for i, row := range rows {
		_, err := db.Exec(`
INSERT INTO scored_commits (
	commitHash, branchName, scoredAt, linesAdded, linesDeleted,
	tabLinesAdded, tabLinesDeleted, composerLinesAdded, composerLinesDeleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
			"commit-"+string(rune('a'+i)),
			"main",
			1000+i,
			row.composerAdded+row.tabAdded,
			row.composerDeleted+row.tabDeleted,
			row.tabAdded,
			row.tabDeleted,
			row.composerAdded,
			row.composerDeleted,
		)
		if err != nil {
			t.Fatalf("inserting fixture row %d: %v", i, err)
		}
	}

	return dbPath
}

func TestDailyCommand_JSONOutput_DefaultGroupByCLI(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "daily", "--json", "--all", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 2 {
		t.Fatalf("expected 2 daily entries, got %d: %s", len(daily), output)
	}

	// Default grouping is by cli.
	totalSessions := 0
	totalTokens := 0
	for _, d := range daily {
		totalSessions += d.Sessions
		totalTokens += d.TokenUsage.Total()
		if d.GroupBy != "cli" {
			t.Errorf("expected group_by %q, got %q", "cli", d.GroupBy)
		}
		if d.Group != "kimi" {
			t.Errorf("expected cli group %q, got %q", "kimi", d.Group)
		}
	}

	if totalSessions != 2 {
		t.Errorf("expected 2 total sessions, got %d", totalSessions)
	}

	// Session1: 450+225+900+60=1635, Session2: 1000+500+2000+110=3610
	expectedTotal := 1635 + 3610
	if totalTokens != expectedTotal {
		t.Errorf("expected %d total tokens, got %d", expectedTotal, totalTokens)
	}
}

func TestCursorActivityCommand_JSONOutput(t *testing.T) {
	bin := buildBinary(t)
	dbPath := writeCursorActivityDB(t, []activityFixtureRow{
		{composerAdded: 9, composerDeleted: 2, tabAdded: 4, tabDeleted: 1},
	})

	output := runCodetok(t, bin, "cursor", "activity", "--json", "--db-path", dbPath)

	var result cursorapi.ActivityResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse activity JSON output: %v\noutput: %s", err, output)
	}

	if !result.HasData {
		t.Fatal("expected activity result to report data")
	}
	if result.ScoredCommits != 1 {
		t.Fatalf("ScoredCommits = %d, want 1", result.ScoredCommits)
	}
	if result.Composer.LinesAdded != 9 || result.Composer.LinesDeleted != 2 {
		t.Fatalf("composer metrics = %+v, want added=9 deleted=2", result.Composer)
	}
	if result.Tab.LinesAdded != 4 || result.Tab.LinesDeleted != 1 {
		t.Fatalf("tab metrics = %+v, want added=4 deleted=1", result.Tab)
	}
}

func TestCursorActivityDoesNotPolluteDailyJSONTokenFields(t *testing.T) {
	bin := buildBinary(t)
	cursorDir := cursorTestdataDir(t)
	dbPath := writeCursorActivityDB(t, []activityFixtureRow{
		{composerAdded: 12, composerDeleted: 3, tabAdded: 6, tabDeleted: 1},
	})

	activityOutput := runCodetok(t, bin, "cursor", "activity", "--json", "--db-path", dbPath)
	if !strings.Contains(activityOutput, "\"composer\"") || !strings.Contains(activityOutput, "\"tab\"") {
		t.Fatalf("activity output = %q, want composer/tab fields", activityOutput)
	}

	args := []string{
		"--claude-dir", emptyDir(t),
		"--codex-dir", emptyDir(t),
		"--kimi-dir", emptyDir(t),
		"daily", "--json", "--all",
		"--cursor-dir", cursorDir,
	}
	output := runCodetok(t, bin, args...)

	var dailyRows []map[string]any
	if err := json.Unmarshal([]byte(output), &dailyRows); err != nil {
		t.Fatalf("failed to parse daily JSON output: %v\noutput: %s", err, output)
	}
	if len(dailyRows) != 2 {
		t.Fatalf("expected 2 daily rows, got %d", len(dailyRows))
	}
	if strings.Contains(output, "\"composer\"") || strings.Contains(output, "\"tab\"") {
		t.Fatalf("daily JSON should not contain activity fields: %s", output)
	}

	for _, row := range dailyRows {
		tokenUsage, ok := row["token_usage"].(map[string]any)
		if !ok {
			t.Fatalf("token_usage field missing or wrong type: %#v", row["token_usage"])
		}
		if len(tokenUsage) != 4 {
			t.Fatalf("token_usage keys = %v, want exactly 4 token fields", tokenUsage)
		}
		for _, key := range []string{"input_other", "output", "input_cache_read", "input_cache_creation"} {
			if _, ok := tokenUsage[key]; !ok {
				t.Fatalf("token_usage missing key %q: %v", key, tokenUsage)
			}
		}
	}
}

func TestCursorActivityDoesNotPolluteSessionJSONTokenFields(t *testing.T) {
	bin := buildBinary(t)
	cursorDir := cursorTestdataDir(t)
	dbPath := writeCursorActivityDB(t, []activityFixtureRow{
		{composerAdded: 7, composerDeleted: 2, tabAdded: 5, tabDeleted: 1},
	})

	activityOutput := runCodetok(t, bin, "cursor", "activity", "--json", "--db-path", dbPath)
	if !strings.Contains(activityOutput, "\"composer\"") || !strings.Contains(activityOutput, "\"tab\"") {
		t.Fatalf("activity output = %q, want composer/tab fields", activityOutput)
	}

	args := []string{
		"--claude-dir", emptyDir(t),
		"--codex-dir", emptyDir(t),
		"--kimi-dir", emptyDir(t),
		"session", "--json",
		"--cursor-dir", cursorDir,
	}
	output := runCodetok(t, bin, args...)

	var sessions []map[string]any
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse session JSON output: %v\noutput: %s", err, output)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if strings.Contains(output, "\"composer\"") || strings.Contains(output, "\"tab\"") {
		t.Fatalf("session JSON should not contain activity fields: %s", output)
	}

	for _, session := range sessions {
		tokenUsage, ok := session["token_usage"].(map[string]any)
		if !ok {
			t.Fatalf("token_usage field missing or wrong type: %#v", session["token_usage"])
		}
		if len(tokenUsage) != 4 {
			t.Fatalf("token_usage keys = %v, want exactly 4 token fields", tokenUsage)
		}
		for _, key := range []string{"input_other", "output", "input_cache_read", "input_cache_creation"} {
			if _, ok := tokenUsage[key]; !ok {
				t.Fatalf("token_usage missing key %q: %v", key, tokenUsage)
			}
		}
	}
}

func TestDailyCommand_JSONOutput_GroupByModel(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "daily", "--json", "--all", "--group-by", "model", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 2 {
		t.Fatalf("expected 2 daily entries, got %d: %s", len(daily), output)
	}

	totalSessions := 0
	totalTokens := 0
	for _, d := range daily {
		totalSessions += d.Sessions
		totalTokens += d.TokenUsage.Total()
		if d.GroupBy != "model" {
			t.Errorf("expected group_by %q, got %q", "model", d.GroupBy)
		}
		if d.Group != "unknown (kimi)" {
			t.Errorf("expected model group %q, got %q", "unknown (kimi)", d.Group)
		}
	}

	if totalSessions != 2 {
		t.Errorf("expected 2 total sessions, got %d", totalSessions)
	}
	expectedTotal := 1635 + 3610
	if totalTokens != expectedTotal {
		t.Errorf("expected %d total tokens, got %d", expectedTotal, totalTokens)
	}
}

func TestSessionCommand_JSONOutput(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "session", "--json", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	var sessions []struct {
		SessionID    string              `json:"session_id"`
		ProviderName string              `json:"provider"`
		Title        string              `json:"title"`
		Date         string              `json:"date"`
		Turns        int                 `json:"turns"`
		TokenUsage   provider.TokenUsage `json:"token_usage"`
	}
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Verify session IDs exist and provider is set
	ids := map[string]bool{}
	for _, s := range sessions {
		ids[s.SessionID] = true
		if s.ProviderName != "kimi" {
			t.Errorf("expected provider %q, got %q", "kimi", s.ProviderName)
		}
	}
	if !ids["uuid-1"] || !ids["uuid-2"] {
		t.Errorf("expected sessions uuid-1 and uuid-2, got %v", ids)
	}
}

func TestDailyCommand_DashboardOutput_DefaultCLI(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "daily", "--all", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	// Verify three-section dashboard
	if !strings.Contains(output, "Daily Total Trend") {
		t.Error("dashboard output missing trend section header")
	}
	if !strings.Contains(output, "CLI Total Ranking") {
		t.Error("dashboard output missing CLI ranking section header")
	}
	if !strings.Contains(output, "Top 5 CLI Share") {
		t.Error("dashboard output missing CLI top share section header")
	}
	if !strings.Contains(output, "Coverage:") {
		t.Error("dashboard output missing coverage line")
	}
	if !strings.Contains(output, "Bar") {
		t.Error("dashboard output missing bar row")
	}

	// Default group-by=cli should show provider group for testdata.
	if !strings.Contains(output, "kimi") {
		t.Error("dashboard output missing default cli group 'kimi'")
	}
}

func TestDailyCommand_DashboardOutput_GroupByModel(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "daily", "--all", "--group-by", "model", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	if !strings.Contains(output, "Model Total Ranking") {
		t.Error("dashboard output missing Model ranking section header")
	}
	if !strings.Contains(output, "Top 5 Model Share") {
		t.Error("dashboard output missing Model top share section header")
	}
	if !strings.Contains(output, "unknown (kimi)") {
		t.Error("dashboard output missing unknown (kimi) model group")
	}
}

func TestDailyCommand_DashboardOutput_TopFlag(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "daily", "--all", "--top", "1", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	if !strings.Contains(output, "Top 1 CLI Share") {
		t.Error("dashboard output missing custom top share section header")
	}
	if strings.Contains(output, "Top 5 CLI Share") {
		t.Error("dashboard output should not show default top value when --top is set")
	}
}

func TestDailyCommand_DashboardOutput_UnitFlag(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)

	outputK := runCodetok(t, bin, isolatedArgs(t, "daily", "--all", "--unit", "k", "--kimi-dir", baseDir)...)
	if !strings.Contains(outputK, "Total(k)") {
		t.Errorf("expected Total(k) header, got:\n%s", outputK)
	}
	if !strings.Contains(outputK, "3.61k") {
		t.Errorf("expected scaled token value 3.61k, got:\n%s", outputK)
	}

	outputRaw := runCodetok(t, bin, isolatedArgs(t, "daily", "--all", "--unit", "raw", "--kimi-dir", baseDir)...)
	if strings.Contains(outputRaw, "Total(k)") {
		t.Errorf("expected raw header without unit suffix, got:\n%s", outputRaw)
	}
	if !strings.Contains(outputRaw, "3610") {
		t.Errorf("expected raw token value 3610, got:\n%s", outputRaw)
	}
}

func TestSessionCommand_TableOutput(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "session", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	// Verify header columns
	if !strings.Contains(output, "Date") {
		t.Error("table output missing Date header")
	}
	if !strings.Contains(output, "Provider") {
		t.Error("table output missing Provider header")
	}
	if !strings.Contains(output, "Session") {
		t.Error("table output missing Session header")
	}
	if !strings.Contains(output, "Title") {
		t.Error("table output missing Title header")
	}

	// Verify TOTAL summary row
	if !strings.Contains(output, "TOTAL") {
		t.Error("table output missing TOTAL summary row")
	}

	// Verify session titles appear
	if !strings.Contains(output, "Implement feature X") {
		t.Error("table output missing session title 'Implement feature X'")
	}

	// Verify provider name appears
	if !strings.Contains(output, "kimi") {
		t.Error("table output missing kimi provider name")
	}

	// Should have at least header + 2 data rows + TOTAL = 4 lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines in table output, got %d:\n%s", len(lines), output)
	}
}

func TestDailyCommand_ProviderFilter(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)

	// Filter by kimi should return results
	args := isolatedArgs(t, "daily", "--json", "--all", "--provider", "kimi", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)
	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}
	if len(daily) != 2 {
		t.Fatalf("expected 2 daily entries with kimi filter, got %d", len(daily))
	}

	// Filter by nonexistent provider should return empty
	output = runCodetok(t, bin, "daily", "--json", "--provider", "nonexistent")
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}
	if len(daily) != 0 {
		t.Errorf("expected 0 daily entries with nonexistent filter, got %d", len(daily))
	}
}

// Ensure testdata directory exists
func TestTestdataExists(t *testing.T) {
	dir := testdataDir(t)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("testdata directory does not exist: %s", dir)
	}
}

func TestClaudeSubagentSessions_JSONOutput(t *testing.T) {
	bin := buildBinary(t)
	claudeDir := claudeTestdataDir(t)
	empty := emptyDir(t)
	args := []string{"--claude-dir", claudeDir, "--codex-dir", empty, "--cursor-dir", empty, "--kimi-dir", empty, "session", "--json"}
	output := runCodetok(t, bin, args...)

	var sessions []struct {
		SessionID    string              `json:"session_id"`
		ProviderName string              `json:"provider"`
		ModelName    string              `json:"model"`
		TokenUsage   provider.TokenUsage `json:"token_usage"`
	}
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	// Should have 2 sessions: main + subagent
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions (main + subagent), got %d: %s", len(sessions), output)
	}

	// Verify both are from claude provider
	for _, s := range sessions {
		if s.ProviderName != "claude" {
			t.Errorf("expected provider %q, got %q", "claude", s.ProviderName)
		}
	}

	// Verify total tokens: main(100+50+200+30=380) + sub(50+10+80+20=160) = 540
	totalTokens := 0
	for _, s := range sessions {
		totalTokens += s.TokenUsage.Total()
	}
	if totalTokens != 540 {
		t.Errorf("expected 540 total tokens, got %d", totalTokens)
	}
}

func TestClaudeSubagentSessions_DailyOutput(t *testing.T) {
	bin := buildBinary(t)
	claudeDir := claudeTestdataDir(t)
	empty := emptyDir(t)
	args := []string{"--claude-dir", claudeDir, "--codex-dir", empty, "--cursor-dir", empty, "--kimi-dir", empty, "daily", "--json", "--all"}
	output := runCodetok(t, bin, args...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 1 {
		t.Fatalf("expected 1 daily entry, got %d: %s", len(daily), output)
	}

	// 2 sessions on the same day
	if daily[0].Sessions != 2 {
		t.Errorf("expected 2 sessions in daily, got %d", daily[0].Sessions)
	}
	if daily[0].TokenUsage.Total() != 540 {
		t.Errorf("expected 540 total tokens, got %d", daily[0].TokenUsage.Total())
	}
}

func TestDailyCommand_JSONOutput_CursorProvider(t *testing.T) {
	bin := buildBinary(t)
	cursorDir := cursorTestdataDir(t)
	empty := emptyDir(t)
	args := []string{
		"daily", "--json", "--all",
		"--cursor-dir", cursorDir,
		"--kimi-dir", empty,
		"--claude-dir", empty,
		"--codex-dir", empty,
	}
	output := runCodetok(t, bin, args...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 2 {
		t.Fatalf("expected 2 daily entries, got %d: %s", len(daily), output)
	}

	totalSessions := 0
	totalTokens := 0
	for _, day := range daily {
		totalSessions += day.Sessions
		totalTokens += day.TokenUsage.Total()
		if day.ProviderName != "cursor" {
			t.Errorf("expected provider %q, got %q", "cursor", day.ProviderName)
		}
		if day.GroupBy != "cli" {
			t.Errorf("expected group_by %q, got %q", "cli", day.GroupBy)
		}
		if day.Group != "cursor" {
			t.Errorf("expected cli group %q, got %q", "cursor", day.Group)
		}
	}

	if totalSessions != 2 {
		t.Errorf("expected 2 total sessions, got %d", totalSessions)
	}
	if totalTokens != 233129 {
		t.Errorf("expected 233129 total tokens, got %d", totalTokens)
	}
}

func TestDailyCommand_DashboardOutput_CursorProvider(t *testing.T) {
	bin := buildBinary(t)
	cursorDir := cursorTestdataDir(t)
	empty := emptyDir(t)
	args := []string{
		"daily", "--all",
		"--cursor-dir", cursorDir,
		"--kimi-dir", empty,
		"--claude-dir", empty,
		"--codex-dir", empty,
	}
	output := runCodetok(t, bin, args...)

	if !strings.Contains(output, "CLI Total Ranking") {
		t.Error("dashboard output missing CLI ranking section header")
	}
	if !strings.Contains(output, "cursor") {
		t.Error("dashboard output missing cursor provider group")
	}
}

func TestSessionCommand_JSONOutput_CursorProvider(t *testing.T) {
	bin := buildBinary(t)
	cursorDir := cursorTestdataDir(t)
	empty := emptyDir(t)
	args := []string{
		"session", "--json",
		"--cursor-dir", cursorDir,
		"--kimi-dir", empty,
		"--claude-dir", empty,
		"--codex-dir", empty,
	}
	output := runCodetok(t, bin, args...)

	var sessions []struct {
		SessionID    string              `json:"session_id"`
		ProviderName string              `json:"provider"`
		Title        string              `json:"title"`
		Date         string              `json:"date"`
		Turns        int                 `json:"turns"`
		TokenUsage   provider.TokenUsage `json:"token_usage"`
	}
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	ids := map[string]bool{}
	for _, session := range sessions {
		ids[session.SessionID] = true
		if session.ProviderName != "cursor" {
			t.Errorf("expected provider %q, got %q", "cursor", session.ProviderName)
		}
	}
	if !ids["usage-export:1"] || !ids["usage-export:2"] {
		t.Errorf("expected sessions usage-export:1 and usage-export:2, got %v", ids)
	}
}
