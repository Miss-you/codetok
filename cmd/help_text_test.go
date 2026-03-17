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

func TestSessionHelpMentionsCursorReportsAreLocalOnly(t *testing.T) {
	output := renderCommandHelp(t, sessionCmd)

	if !strings.Contains(output, "never trigger implicit Cursor login or sync") {
		t.Fatalf("session help missing local-only Cursor guidance:\n%s", output)
	}
	if !strings.Contains(output, "imports/") || !strings.Contains(output, "synced/") {
		t.Fatalf("session help missing default Cursor source layout:\n%s", output)
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
