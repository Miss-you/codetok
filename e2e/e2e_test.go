package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	cursorapi "github.com/miss-you/codetok/cursor"
	"github.com/miss-you/codetok/internal/testutil"
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
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
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

func runCodetokWithEnv(t *testing.T, binPath string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %v failed: %v\nstderr: %s", args, err, stderr.String())
	}
	return stdout.String()
}

type activityFixtureRow = testutil.CursorActivityRow

func writeCursorActivityDB(t *testing.T, rows []activityFixtureRow) string {
	return testutil.WriteCursorActivityDB(t, rows)
}

func defaultCursorArgs(t *testing.T, extraArgs ...string) []string {
	t.Helper()
	empty := emptyDir(t)
	base := []string{"--claude-dir", empty, "--codex-dir", empty, "--kimi-dir", empty}
	return append(base, extraArgs...)
}

func cursorEnv(home string) []string {
	return []string{"HOME=" + home}
}

func proxyEnv(proxyURL string) []string {
	return []string{
		"HTTP_PROXY=" + proxyURL,
		"HTTPS_PROXY=" + proxyURL,
		"ALL_PROXY=" + proxyURL,
		"NO_PROXY=",
	}
}

func newProxyTrap(t *testing.T) (string, func() int32) {
	t.Helper()

	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	return server.URL, func() int32 {
		return atomic.LoadInt32(&count)
	}
}

func writeCursorCSVFixtureE2E(t *testing.T, path string, rows ...string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating parent directory: %v", err)
	}

	content := "Date,Kind,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens\n"
	content += strings.Join(rows, "\n")
	content += "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing csv fixture: %v", err)
	}
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
		{ComposerAdded: 9, ComposerDeleted: 2, TabAdded: 4, TabDeleted: 1},
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
		{ComposerAdded: 12, ComposerDeleted: 3, TabAdded: 6, TabDeleted: 1},
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
		{ComposerAdded: 7, ComposerDeleted: 2, TabAdded: 5, TabDeleted: 1},
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

	// Daily event aggregation counts distinct contributing session IDs.
	// Claude subagent files in this fixture share session-main with the parent.
	if daily[0].Sessions != 1 {
		t.Errorf("expected 1 contributing session ID in daily, got %d", daily[0].Sessions)
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

func TestDailyCommand_JSONOutput_CursorDefaultRootImportOnly(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	writeCursorCSVFixtureE2E(t, filepath.Join(home, ".codetok", "cursor", "imports", "manual.csv"),
		`"2026-02-18T10:00:00Z","Included","manual-model","2","3","4","5"`,
	)

	output := runCodetokWithEnv(t, bin, cursorEnv(home), defaultCursorArgs(t, "daily", "--json", "--all", "--provider", "cursor")...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 1 {
		t.Fatalf("expected 1 daily entry, got %d: %s", len(daily), output)
	}
	if daily[0].Sessions != 1 {
		t.Fatalf("expected 1 session, got %d", daily[0].Sessions)
	}
	if daily[0].TokenUsage.Total() != 14 {
		t.Fatalf("expected total tokens 14, got %d", daily[0].TokenUsage.Total())
	}
}

func TestSessionCommand_JSONOutput_CursorDefaultRootSyncOnly(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	writeCursorCSVFixtureE2E(t, filepath.Join(home, ".codetok", "cursor", "synced", "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","3","4","5","6"`,
	)

	output := runCodetokWithEnv(t, bin, cursorEnv(home), defaultCursorArgs(t, "session", "--json", "--provider", "cursor")...)

	var sessions []struct {
		SessionID  string              `json:"session_id"`
		TokenUsage provider.TokenUsage `json:"token_usage"`
	}
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d: %s", len(sessions), output)
	}
	if sessions[0].SessionID != "cached:1" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "cached:1")
	}
	if sessions[0].TokenUsage.Total() != 18 {
		t.Fatalf("expected total tokens 18, got %d", sessions[0].TokenUsage.Total())
	}
}

func TestDailyCommand_JSONOutput_CursorDefaultRootEmpty(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()

	output := runCodetokWithEnv(t, bin, cursorEnv(home), defaultCursorArgs(t, "daily", "--json", "--all", "--provider", "cursor")...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 0 {
		t.Fatalf("expected 0 daily entries for empty Cursor root, got %d: %s", len(daily), output)
	}
}

func TestDailyCommand_JSONOutput_CursorDefaultRootMergesImportSyncAndLegacy(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	root := filepath.Join(home, ".codetok", "cursor")
	writeCursorCSVFixtureE2E(t, filepath.Join(root, "legacy.csv"),
		`"2026-02-17T10:00:00Z","Included","legacy-model","1","2","3","4"`,
	)
	writeCursorCSVFixtureE2E(t, filepath.Join(root, "imports", "manual.csv"),
		`"2026-02-18T10:00:00Z","Included","manual-model","2","3","4","5"`,
	)
	writeCursorCSVFixtureE2E(t, filepath.Join(root, "synced", "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","3","4","5","6"`,
	)
	writeCursorCSVFixtureE2E(t, filepath.Join(root, "archive", "ignored.csv"),
		`"2026-02-20T10:00:00Z","Included","ignored-model","11","12","13","14"`,
	)

	output := runCodetokWithEnv(t, bin, cursorEnv(home), defaultCursorArgs(t, "daily", "--json", "--all", "--provider", "cursor")...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	totalSessions := 0
	totalTokens := 0
	for _, day := range daily {
		totalSessions += day.Sessions
		totalTokens += day.TokenUsage.Total()
	}

	if totalSessions != 3 {
		t.Fatalf("expected 3 sessions, got %d", totalSessions)
	}
	if totalTokens != 42 {
		t.Fatalf("expected total tokens 42, got %d", totalTokens)
	}
}

func TestSessionCommand_JSONOutput_CursorDefaultRootSyncFailureKeepsCachedData(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	root := filepath.Join(home, ".codetok", "cursor", "synced")
	writeCursorCSVFixtureE2E(t, filepath.Join(root, "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","3","4","5","6"`,
	)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("creating synced dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "failed.csv"), []byte("Date,Model\nbroken"), 0o644); err != nil {
		t.Fatalf("writing invalid sync file: %v", err)
	}

	output := runCodetokWithEnv(t, bin, cursorEnv(home), defaultCursorArgs(t, "session", "--json", "--provider", "cursor")...)

	var sessions []struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d: %s", len(sessions), output)
	}
	if sessions[0].SessionID != "cached:1" {
		t.Fatalf("SessionID = %q, want %q", sessions[0].SessionID, "cached:1")
	}
}

func TestDailyCommand_JSONOutput_CursorDirOverrideIsAuthoritative(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	writeCursorCSVFixtureE2E(t, filepath.Join(home, ".codetok", "cursor", "imports", "default.csv"),
		`"2026-02-18T10:00:00Z","Included","default-model","2","3","4","5"`,
	)

	customDir := t.TempDir()
	writeCursorCSVFixtureE2E(t, filepath.Join(customDir, "custom.csv"),
		`"2026-02-21T10:00:00Z","Included","custom-model","7","8","9","10"`,
	)

	args := append(defaultCursorArgs(t, "daily", "--json", "--all", "--provider", "cursor"), "--cursor-dir", customDir)
	output := runCodetokWithEnv(t, bin, cursorEnv(home), args...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 1 {
		t.Fatalf("expected 1 daily entry, got %d: %s", len(daily), output)
	}
	if daily[0].Sessions != 1 {
		t.Fatalf("expected 1 session, got %d", daily[0].Sessions)
	}
	if daily[0].TokenUsage.Total() != 34 {
		t.Fatalf("expected total tokens 34, got %d", daily[0].TokenUsage.Total())
	}
}

func TestDailyCommand_CursorDoesNotAccessRemoteAPI(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	writeCursorCSVFixtureE2E(t, filepath.Join(home, ".codetok", "cursor", "synced", "cached.csv"),
		`"2026-02-19T10:00:00Z","Included","synced-model","3","4","5","6"`,
	)

	proxyURL, requests := newProxyTrap(t)
	env := append(cursorEnv(home), proxyEnv(proxyURL)...)
	output := runCodetokWithEnv(t, bin, env, defaultCursorArgs(t, "daily", "--json", "--all", "--provider", "cursor")...)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}
	if requests() != 0 {
		t.Fatalf("expected 0 proxy requests, got %d", requests())
	}
}

func TestSessionCommand_CursorDoesNotAccessRemoteAPI(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	writeCursorCSVFixtureE2E(t, filepath.Join(home, ".codetok", "cursor", "imports", "manual.csv"),
		`"2026-02-18T10:00:00Z","Included","manual-model","2","3","4","5"`,
	)

	proxyURL, requests := newProxyTrap(t)
	env := append(cursorEnv(home), proxyEnv(proxyURL)...)
	output := runCodetokWithEnv(t, bin, env, defaultCursorArgs(t, "session", "--json", "--provider", "cursor")...)

	var sessions []map[string]any
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d: %s", len(sessions), output)
	}
	if requests() != 0 {
		t.Fatalf("expected 0 proxy requests, got %d", requests())
	}
}
