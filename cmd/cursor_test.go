package cmd

import (
	"context"
	"strings"
	"testing"

	cursorapi "github.com/miss-you/codetok/cursor"
)

type stubCursorCommandService struct {
	loginResult    cursorapi.ValidationResult
	loginErr       error
	statusResult   cursorapi.StatusResult
	statusErr      error
	activityResult cursorapi.ActivityResult
	activityErr    error
	syncResult     cursorapi.SyncResult
	syncErr        error
	logoutErr      error

	loginToken     string
	activityDBPath string
	logoutDone     bool
}

func (s *stubCursorCommandService) Login(_ context.Context, token string) (cursorapi.ValidationResult, error) {
	s.loginToken = token
	return s.loginResult, s.loginErr
}

func (s *stubCursorCommandService) Status(context.Context) (cursorapi.StatusResult, error) {
	return s.statusResult, s.statusErr
}

func (s *stubCursorCommandService) Activity(_ context.Context, dbPath string) (cursorapi.ActivityResult, error) {
	s.activityDBPath = dbPath
	return s.activityResult, s.activityErr
}

func (s *stubCursorCommandService) Sync(context.Context) (cursorapi.SyncResult, error) {
	return s.syncResult, s.syncErr
}

func (s *stubCursorCommandService) Logout() error {
	s.logoutDone = true
	return s.logoutErr
}

func TestNewCursorCommand_IncludesLifecycleSubcommands(t *testing.T) {
	cmd := newCursorCommand(&stubCursorCommandService{})

	got := make(map[string]bool)
	for _, child := range cmd.Commands() {
		got[child.Name()] = true
	}

	for _, name := range []string{"login", "status", "activity", "sync", "logout"} {
		if !got[name] {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}

	if !strings.Contains(cmd.Short, "Cursor") {
		t.Fatalf("cursor command short help = %q, want Cursor help text", cmd.Short)
	}
}

func TestCursorLoginCommand_RequiresTokenFromFlagOrStdin(t *testing.T) {
	cmd := newCursorCommand(&stubCursorCommandService{})
	cmd.SetArgs([]string{"login"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no token is provided")
	}
	if !strings.Contains(err.Error(), "provide a Cursor session token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCursorLoginCommand_UsesTokenFlag(t *testing.T) {
	svc := &stubCursorCommandService{
		loginResult: cursorapi.ValidationResult{Valid: true, MembershipType: "pro"},
	}
	cmd := newCursorCommand(svc)
	cmd.SetArgs([]string{"login", "--token", "token-123"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("login command failed: %v", err)
		}
	})

	if svc.loginToken != "token-123" {
		t.Fatalf("login token = %q, want token-123", svc.loginToken)
	}
	if !strings.Contains(output, "Cursor login successful") {
		t.Fatalf("login output = %q, want success message", output)
	}
}

func TestCursorStatusCommand_PrintsLoggedOutState(t *testing.T) {
	svc := &stubCursorCommandService{
		statusResult: cursorapi.StatusResult{HasCredentials: false},
	}
	cmd := newCursorCommand(svc)
	cmd.SetArgs([]string{"status"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("status command failed: %v", err)
		}
	})

	if !strings.Contains(strings.ToLower(output), "not logged in") {
		t.Fatalf("status output = %q, want logged-out message", output)
	}
}

func TestCursorSyncCommand_PrintsSyncedPath(t *testing.T) {
	svc := &stubCursorCommandService{
		syncResult: cursorapi.SyncResult{Path: "/tmp/cursor/synced/usage.csv", Bytes: 128},
	}
	cmd := newCursorCommand(svc)
	cmd.SetArgs([]string{"sync"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("sync command failed: %v", err)
		}
	})

	if !strings.Contains(output, "/tmp/cursor/synced/usage.csv") {
		t.Fatalf("sync output = %q, want synced path", output)
	}
}

func TestCursorActivityCommand_UsesDBPathOverrideAndPrintsJSON(t *testing.T) {
	svc := &stubCursorCommandService{
		activityResult: cursorapi.ActivityResult{
			DBPath:        "/tmp/cursor/ai-code-tracking.db",
			HasData:       true,
			ScoredCommits: 2,
			Composer:      cursorapi.ActivityMetric{LinesAdded: 9, LinesDeleted: 3},
			Tab:           cursorapi.ActivityMetric{LinesAdded: 4, LinesDeleted: 1},
		},
	}
	cmd := newCursorCommand(svc)
	cmd.SetArgs([]string{"activity", "--json", "--db-path", "/tmp/cursor/ai-code-tracking.db"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("activity command failed: %v", err)
		}
	})

	if svc.activityDBPath != "/tmp/cursor/ai-code-tracking.db" {
		t.Fatalf("activity db path = %q, want /tmp/cursor/ai-code-tracking.db", svc.activityDBPath)
	}
	if !strings.Contains(output, "\"composer\"") || !strings.Contains(output, "\"tab\"") {
		t.Fatalf("activity json output = %q, want composer/tab fields", output)
	}
	if strings.Contains(strings.ToLower(output), "token") {
		t.Fatalf("activity json output should not use token wording: %q", output)
	}
}

func TestCursorActivityCommand_TableOutputUsesAttributionLabels(t *testing.T) {
	svc := &stubCursorCommandService{
		activityResult: cursorapi.ActivityResult{
			DBPath:        "/tmp/cursor/ai-code-tracking.db",
			HasData:       true,
			ScoredCommits: 1,
			Composer:      cursorapi.ActivityMetric{LinesAdded: 8, LinesDeleted: 2},
			Tab:           cursorapi.ActivityMetric{LinesAdded: 0, LinesDeleted: 0},
		},
	}
	cmd := newCursorCommand(svc)
	cmd.SetArgs([]string{"activity"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("activity command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Cursor Activity Attribution") {
		t.Fatalf("activity output = %q, want attribution heading", output)
	}
	if !strings.Contains(output, "composer") || !strings.Contains(output, "tab") {
		t.Fatalf("activity output = %q, want composer/tab rows", output)
	}
	if strings.Contains(strings.ToLower(output), "token") {
		t.Fatalf("activity output should not use token wording: %q", output)
	}
}

func TestCursorLogoutCommand_CallsService(t *testing.T) {
	svc := &stubCursorCommandService{}
	cmd := newCursorCommand(svc)
	cmd.SetArgs([]string{"logout"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("logout command failed: %v", err)
		}
	})

	if !svc.logoutDone {
		t.Fatal("expected logout service to be called")
	}
	if !strings.Contains(strings.ToLower(output), "logged out") {
		t.Fatalf("logout output = %q, want logged-out message", output)
	}
}
