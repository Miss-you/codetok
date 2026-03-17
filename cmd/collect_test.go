package cmd

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/miss-you/codetok/provider"
)

type collectTestProvider struct {
	name     string
	sessions []provider.SessionInfo
	err      error
	seenDirs []string
}

func (p *collectTestProvider) Name() string {
	return p.name
}

func (p *collectTestProvider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	p.seenDirs = append(p.seenDirs, baseDir)
	if p.err != nil {
		return nil, p.err
	}
	return p.sessions, nil
}

func TestCollectSessionsFromProviders_UsesBaseDirAndProviderOverride(t *testing.T) {
	alpha := &collectTestProvider{
		name:     "alpha",
		sessions: []provider.SessionInfo{{SessionID: "alpha-1", ProviderName: "alpha"}},
	}
	beta := &collectTestProvider{
		name:     "beta",
		sessions: []provider.SessionInfo{{SessionID: "beta-1", ProviderName: "beta"}},
	}
	cmd := newCollectTestCommand("alpha", "beta")
	if err := cmd.Flags().Set("base-dir", "/shared"); err != nil {
		t.Fatalf("setting --base-dir: %v", err)
	}
	if err := cmd.Flags().Set("beta-dir", "/beta-only"); err != nil {
		t.Fatalf("setting --beta-dir: %v", err)
	}

	sessions, err := collectSessionsFromProviders(cmd, []provider.Provider{alpha, beta})
	if err != nil {
		t.Fatalf("collectSessionsFromProviders returned error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if len(alpha.seenDirs) != 1 || alpha.seenDirs[0] != "/shared" {
		t.Fatalf("alpha dirs = %v, want [/shared]", alpha.seenDirs)
	}
	if len(beta.seenDirs) != 1 || beta.seenDirs[0] != "/beta-only" {
		t.Fatalf("beta dirs = %v, want [/beta-only]", beta.seenDirs)
	}
}

func TestCollectSessionsFromProviders_RespectsProviderFilter(t *testing.T) {
	alpha := &collectTestProvider{name: "alpha"}
	beta := &collectTestProvider{
		name:     "beta",
		sessions: []provider.SessionInfo{{SessionID: "beta-1", ProviderName: "beta"}},
	}
	cmd := newCollectTestCommand("alpha", "beta")
	if err := cmd.Flags().Set("provider", "beta"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	sessions, err := collectSessionsFromProviders(cmd, []provider.Provider{alpha, beta})
	if err != nil {
		t.Fatalf("collectSessionsFromProviders returned error: %v", err)
	}

	if len(sessions) != 1 || sessions[0].SessionID != "beta-1" {
		t.Fatalf("sessions = %#v, want only beta", sessions)
	}
	if len(alpha.seenDirs) != 0 {
		t.Fatalf("alpha should not be collected when filtered, seen %v", alpha.seenDirs)
	}
	if len(beta.seenDirs) != 1 {
		t.Fatalf("beta should be collected exactly once, seen %v", beta.seenDirs)
	}
}

func TestCollectSessionsFromProviders_SkipsMissingDirectories(t *testing.T) {
	missing := &collectTestProvider{name: "missing", err: os.ErrNotExist}
	ok := &collectTestProvider{
		name:     "ok",
		sessions: []provider.SessionInfo{{SessionID: "ok-1", ProviderName: "ok"}},
	}
	cmd := newCollectTestCommand("missing", "ok")

	sessions, err := collectSessionsFromProviders(cmd, []provider.Provider{missing, ok})
	if err != nil {
		t.Fatalf("collectSessionsFromProviders returned error: %v", err)
	}

	if len(sessions) != 1 || sessions[0].SessionID != "ok-1" {
		t.Fatalf("sessions = %#v, want only ok provider data", sessions)
	}
}

func TestCollectSessionsFromProviders_ReturnsProviderErrors(t *testing.T) {
	boom := errors.New("boom")
	bad := &collectTestProvider{name: "bad", err: boom}
	cmd := newCollectTestCommand("bad")

	_, err := collectSessionsFromProviders(cmd, []provider.Provider{bad})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "collecting sessions from bad") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("wrapped error should contain root cause, got: %v", err)
	}
}

func newCollectTestCommand(providerNames ...string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("base-dir", "", "")
	for _, name := range providerNames {
		cmd.Flags().String(providerDirFlag(name), "", "")
	}
	return cmd
}
