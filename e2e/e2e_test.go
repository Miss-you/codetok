package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

// emptyDir returns a path to an empty temp directory (to isolate providers in tests).
func emptyDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// isolatedArgs returns args that point all providers to empty dirs except the one being tested.
func isolatedArgs(t *testing.T, extraArgs ...string) []string {
	t.Helper()
	empty := emptyDir(t)
	base := []string{"--claude-dir", empty, "--codex-dir", empty}
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

func TestDailyCommand_JSONOutput(t *testing.T) {
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

	// Verify each day has correct session count and provider
	totalSessions := 0
	totalTokens := 0
	for _, d := range daily {
		totalSessions += d.Sessions
		totalTokens += d.TokenUsage.Total()
		if d.ProviderName != "kimi" {
			t.Errorf("expected provider %q, got %q", "kimi", d.ProviderName)
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

func TestDailyCommand_TableOutput(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	args := isolatedArgs(t, "daily", "--all", "--kimi-dir", baseDir)
	output := runCodetok(t, bin, args...)

	// Verify header
	if !strings.Contains(output, "Date") {
		t.Error("table output missing Date header")
	}
	if !strings.Contains(output, "Provider") {
		t.Error("table output missing Provider header")
	}
	if !strings.Contains(output, "Sessions") {
		t.Error("table output missing Sessions header")
	}
	if !strings.Contains(output, "Total") {
		t.Error("table output missing Total header")
	}

	// Verify TOTAL summary row
	if !strings.Contains(output, "TOTAL") {
		t.Error("table output missing TOTAL summary row")
	}

	// Verify provider name appears in data rows
	if !strings.Contains(output, "kimi") {
		t.Error("table output missing kimi provider name")
	}

	// Should have at least header + 2 data rows + TOTAL row = 4 lines minimum
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines in table output, got %d:\n%s", len(lines), output)
	}
}

func TestDailyCommand_TableOutput_UnitFlag(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)

	outputK := runCodetok(t, bin, isolatedArgs(t, "daily", "--all", "--unit", "k", "--kimi-dir", baseDir)...)
	if !strings.Contains(outputK, "Input(k)") {
		t.Errorf("expected Input(k) header, got:\n%s", outputK)
	}
	if !strings.Contains(outputK, "3.61k") {
		t.Errorf("expected scaled token value 3.61k, got:\n%s", outputK)
	}

	outputRaw := runCodetok(t, bin, isolatedArgs(t, "daily", "--all", "--unit", "raw", "--kimi-dir", baseDir)...)
	if !strings.Contains(outputRaw, "Input") || strings.Contains(outputRaw, "Input(k)") {
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
