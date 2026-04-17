package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDailyHelpMentionsCursorReportsAreLocalOnly(t *testing.T) {
	output := renderCommandHelp(t, dailyCmd)

	if !strings.Contains(output, "never trigger implicit Cursor login or sync") {
		t.Fatalf("daily help missing local-only Cursor guidance:\n%s", output)
	}
	if !strings.Contains(output, "imports/") || !strings.Contains(output, "synced/") {
		t.Fatalf("daily help missing default Cursor source layout:\n%s", output)
	}
}

func TestDailyHelpMentionsEventDateTimezoneAndCodexHome(t *testing.T) {
	output := renderCommandHelp(t, dailyCmd)

	if !strings.Contains(output, "usage events by event date") {
		t.Fatalf("daily help missing event-date aggregation guidance:\n%s", output)
	}
	if !strings.Contains(output, "IANA timezone name") || !strings.Contains(output, "local timezone") {
		t.Fatalf("daily help missing timezone guidance:\n%s", output)
	}
	if !strings.Contains(output, "$CODEX_HOME/sessions") || !strings.Contains(output, "~/.codex/sessions") {
		t.Fatalf("daily help missing Codex source resolution guidance:\n%s", output)
	}
}

func TestSessionHelpMentionsCursorReportsAreLocalOnly(t *testing.T) {
	output := renderCommandHelp(t, sessionCmd)

	if !strings.Contains(output, "never trigger implicit Cursor login or sync") {
		t.Fatalf("session help missing local-only Cursor guidance:\n%s", output)
	}
	if !strings.Contains(output, "imports/") || !strings.Contains(output, "synced/") {
		t.Fatalf("session help missing default Cursor source layout:\n%s", output)
	}
}

func TestSessionHelpMentionsEventFilteringTimezoneAndCodexHome(t *testing.T) {
	output := renderCommandHelp(t, sessionCmd)

	if !strings.Contains(output, "usage events in the selected date range") {
		t.Fatalf("session help missing usage-event filtering guidance:\n%s", output)
	}
	if !strings.Contains(output, "IANA timezone name") || !strings.Contains(output, "local timezone") {
		t.Fatalf("session help missing timezone guidance:\n%s", output)
	}
	if !strings.Contains(output, "$CODEX_HOME/sessions") || !strings.Contains(output, "~/.codex/sessions") {
		t.Fatalf("session help missing Codex source resolution guidance:\n%s", output)
	}
}

func renderCommandHelp(t *testing.T, cmd *cobra.Command) string {
	t.Helper()

	var buf bytes.Buffer
	oldOut := cmd.OutOrStdout()
	oldErr := cmd.ErrOrStderr()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	t.Cleanup(func() {
		cmd.SetOut(oldOut)
		cmd.SetErr(oldErr)
	})

	if err := cmd.Help(); err != nil {
		t.Fatalf("rendering help: %v", err)
	}
	return buf.String()
}
