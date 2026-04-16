package cmd

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

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

type collectTestUsageEventProvider struct {
	collectTestProvider
	events        []provider.UsageEvent
	eventErr      error
	seenEventDirs []string
}

func (p *collectTestUsageEventProvider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error) {
	p.seenEventDirs = append(p.seenEventDirs, baseDir)
	if p.eventErr != nil {
		return nil, p.eventErr
	}
	return p.events, nil
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

func TestCollectUsageEventsFromProviders_UsesNativeEvents(t *testing.T) {
	timestamp := time.Date(2026, 4, 16, 10, 30, 0, 0, time.UTC)
	nativeEvent := provider.UsageEvent{
		ProviderName: "source-provider",
		ModelName:    "gpt-5.4",
		SessionID:    "native-session",
		Title:        "Native title",
		WorkDirHash:  "abc123",
		Timestamp:    timestamp,
		TokenUsage: provider.TokenUsage{
			InputOther:       100,
			Output:           20,
			InputCacheRead:   30,
			InputCacheCreate: 40,
		},
		SourcePath: "/logs/native.jsonl",
		EventID:    "event-1",
	}
	native := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{
			name: "native",
			err:  errors.New("sessions should not be collected for native events"),
		},
		events: []provider.UsageEvent{nativeEvent},
	}
	cmd := newCollectTestCommand("native")
	if err := cmd.Flags().Set("base-dir", "/shared"); err != nil {
		t.Fatalf("setting --base-dir: %v", err)
	}
	if err := cmd.Flags().Set("native-dir", "/native-only"); err != nil {
		t.Fatalf("setting --native-dir: %v", err)
	}

	events, err := collectUsageEventsFromProviders(cmd, []provider.Provider{native})
	if err != nil {
		t.Fatalf("collectUsageEventsFromProviders returned error: %v", err)
	}

	if !reflect.DeepEqual(events, []provider.UsageEvent{nativeEvent}) {
		t.Fatalf("events = %#v, want native event unchanged", events)
	}
	if len(native.seenEventDirs) != 1 || native.seenEventDirs[0] != "/native-only" {
		t.Fatalf("native event dirs = %v, want [/native-only]", native.seenEventDirs)
	}
	if len(native.seenDirs) != 0 {
		t.Fatalf("native provider should not fall back to sessions, seen %v", native.seenDirs)
	}
}

func TestCollectUsageEventsFromProviders_FallsBackToSessionEvent(t *testing.T) {
	start := time.Date(2026, 4, 16, 11, 45, 0, 0, time.UTC)
	session := provider.SessionInfo{
		ProviderName: "legacy",
		ModelName:    "legacy-model",
		SessionID:    "legacy-session",
		Title:        "Legacy title",
		WorkDirHash:  "def456",
		StartTime:    start,
		EndTime:      start.Add(10 * time.Minute),
		Turns:        3,
		TokenUsage: provider.TokenUsage{
			InputOther:       200,
			Output:           50,
			InputCacheRead:   60,
			InputCacheCreate: 70,
		},
	}
	legacy := &collectTestProvider{
		name:     "legacy",
		sessions: []provider.SessionInfo{session},
	}
	cmd := newCollectTestCommand("legacy")
	if err := cmd.Flags().Set("base-dir", "/shared"); err != nil {
		t.Fatalf("setting --base-dir: %v", err)
	}

	events, err := collectUsageEventsFromProviders(cmd, []provider.Provider{legacy})
	if err != nil {
		t.Fatalf("collectUsageEventsFromProviders returned error: %v", err)
	}

	want := []provider.UsageEvent{{
		ProviderName: "legacy",
		ModelName:    "legacy-model",
		SessionID:    "legacy-session",
		Title:        "Legacy title",
		WorkDirHash:  "def456",
		Timestamp:    start,
		TokenUsage:   session.TokenUsage,
	}}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
	if len(legacy.seenDirs) != 1 || legacy.seenDirs[0] != "/shared" {
		t.Fatalf("legacy dirs = %v, want [/shared]", legacy.seenDirs)
	}
}

func TestCollectUsageEventsFromProviders_RespectsProviderFilter(t *testing.T) {
	alpha := &collectTestProvider{name: "alpha"}
	beta := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "beta"},
		events: []provider.UsageEvent{{
			ProviderName: "beta",
			SessionID:    "beta-event",
		}},
	}
	cmd := newCollectTestCommand("alpha", "beta")
	if err := cmd.Flags().Set("provider", "beta"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	events, err := collectUsageEventsFromProviders(cmd, []provider.Provider{alpha, beta})
	if err != nil {
		t.Fatalf("collectUsageEventsFromProviders returned error: %v", err)
	}

	if len(events) != 1 || events[0].SessionID != "beta-event" {
		t.Fatalf("events = %#v, want only beta", events)
	}
	if len(alpha.seenDirs) != 0 {
		t.Fatalf("alpha should not be collected when filtered, seen %v", alpha.seenDirs)
	}
	if len(beta.seenEventDirs) != 1 {
		t.Fatalf("beta should be collected exactly once, seen %v", beta.seenEventDirs)
	}
}

func TestCollectUsageEventsFromProviders_SkipsMissingDirectories(t *testing.T) {
	missingNative := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "missing-native"},
		eventErr:            os.ErrNotExist,
	}
	missingLegacy := &collectTestProvider{name: "missing-legacy", err: os.ErrNotExist}
	ok := &collectTestProvider{
		name: "ok",
		sessions: []provider.SessionInfo{{
			ProviderName: "ok",
			SessionID:    "ok-session",
		}},
	}
	cmd := newCollectTestCommand("missing-native", "missing-legacy", "ok")

	events, err := collectUsageEventsFromProviders(cmd, []provider.Provider{missingNative, missingLegacy, ok})
	if err != nil {
		t.Fatalf("collectUsageEventsFromProviders returned error: %v", err)
	}

	if len(events) != 1 || events[0].SessionID != "ok-session" {
		t.Fatalf("events = %#v, want only ok provider data", events)
	}
}

func TestCollectUsageEventsFromProviders_ReturnsNativeProviderErrors(t *testing.T) {
	boom := errors.New("boom")
	bad := &collectTestUsageEventProvider{
		collectTestProvider: collectTestProvider{name: "bad"},
		eventErr:            boom,
	}
	cmd := newCollectTestCommand("bad")

	_, err := collectUsageEventsFromProviders(cmd, []provider.Provider{bad})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "collecting usage events from bad") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("wrapped error should contain root cause, got: %v", err)
	}
}

func TestCollectUsageEventsFromProviders_ReturnsSessionFallbackErrors(t *testing.T) {
	boom := errors.New("boom")
	bad := &collectTestProvider{name: "bad", err: boom}
	cmd := newCollectTestCommand("bad")

	_, err := collectUsageEventsFromProviders(cmd, []provider.Provider{bad})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "collecting sessions for usage events from bad") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("wrapped error should contain root cause, got: %v", err)
	}
}

func TestCollectUsageEvents_UsesRegisteredProviders(t *testing.T) {
	cmd := newCollectTestCommand()
	if err := cmd.Flags().Set("provider", "nonexistent"); err != nil {
		t.Fatalf("setting --provider: %v", err)
	}

	events, err := collectUsageEvents(cmd)
	if err != nil {
		t.Fatalf("collectUsageEvents returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v, want none for unmatched provider", events)
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
