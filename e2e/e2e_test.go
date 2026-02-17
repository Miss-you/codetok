package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Miss-you/codetok/provider"
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
	output := runCodetok(t, bin, "daily", "--json", "--base-dir", baseDir)

	var daily []provider.DailyStats
	if err := json.Unmarshal([]byte(output), &daily); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(daily) != 2 {
		t.Fatalf("expected 2 daily entries, got %d: %s", len(daily), output)
	}

	// Verify each day has correct session count
	totalSessions := 0
	totalTokens := 0
	for _, d := range daily {
		totalSessions += d.Sessions
		totalTokens += d.TokenUsage.Total()
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
	output := runCodetok(t, bin, "session", "--json", "--base-dir", baseDir)

	var sessions []struct {
		SessionID  string              `json:"session_id"`
		Title      string              `json:"title"`
		Date       string              `json:"date"`
		Turns      int                 `json:"turns"`
		TokenUsage provider.TokenUsage `json:"token_usage"`
	}
	if err := json.Unmarshal([]byte(output), &sessions); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Verify session IDs exist
	ids := map[string]bool{}
	for _, s := range sessions {
		ids[s.SessionID] = true
	}
	if !ids["uuid-1"] || !ids["uuid-2"] {
		t.Errorf("expected sessions uuid-1 and uuid-2, got %v", ids)
	}
}

func TestDailyCommand_TableOutput(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	output := runCodetok(t, bin, "daily", "--base-dir", baseDir)

	// Verify header
	if !strings.Contains(output, "Date") {
		t.Error("table output missing Date header")
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

	// Should have at least header + 2 data rows + TOTAL row = 4 lines minimum
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines in table output, got %d:\n%s", len(lines), output)
	}
}

func TestSessionCommand_TableOutput(t *testing.T) {
	bin := buildBinary(t)
	baseDir := testdataDir(t)
	output := runCodetok(t, bin, "session", "--base-dir", baseDir)

	// Verify header columns
	if !strings.Contains(output, "Date") {
		t.Error("table output missing Date header")
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

	// Should have at least header + 2 data rows + TOTAL = 4 lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines in table output, got %d:\n%s", len(lines), output)
	}
}

// Ensure testdata directory exists
func TestTestdataExists(t *testing.T) {
	dir := testdataDir(t)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("testdata directory does not exist: %s", dir)
	}
}
