package kimi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/miss-you/codetok/provider"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements provider.Provider for Kimi Code.
type Provider struct{}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "kimi"
}

// wireEvent represents a single line in a Kimi Code wire.jsonl file.
type wireEvent struct {
	Type   string          `json:"type"`
	Model  string          `json:"model"`
	Usage  *wireTokenUsage `json:"usage"`
	Origin *struct {
		Kind string `json:"kind"`
	} `json:"origin"`
	Time int64 `json:"time"` // milliseconds since epoch
}

// wireTokenUsage is the usage payload of a usage.record event.
type wireTokenUsage struct {
	InputOther         int `json:"inputOther"`
	Output             int `json:"output"`
	InputCacheRead     int `json:"inputCacheRead"`
	InputCacheCreation int `json:"inputCacheCreation"`
}

// sessionState mirrors the state.json file stored in each session directory.
type sessionState struct {
	Title string `json:"title"`
}

// sessionRef points at one session directory and its per-agent wire files.
type sessionRef struct {
	dir       string
	wirePaths []string
}

// CollectSessions scans baseDir for Kimi Code session directories and returns session info.
// The expected directory layout is: baseDir/<work-dir>/<session-id>/agents/<agent>/wire.jsonl
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	if baseDir == "" {
		baseDir = defaultKimiSessionsDir()
	}

	refs, err := discoverSessions(baseDir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(refs))
	for i, ref := range refs {
		paths[i] = ref.dir
	}
	sessions := provider.ParseParallel(paths, 0, parseSession)

	return sessions, nil
}

// CollectUsageEvents scans baseDir for Kimi Code session directories and returns native usage events.
func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error) {
	return p.collectUsageEvents(baseDir, provider.UsageEventCollectOptions{})
}

func (p *Provider) CollectUsageEventsInRange(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	return p.collectUsageEvents(baseDir, opts)
}

func (p *Provider) collectUsageEvents(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	if baseDir == "" {
		baseDir = defaultKimiSessionsDir()
	}

	refs, err := discoverSessions(baseDir)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, ref := range refs {
		for _, wirePath := range ref.wirePaths {
			info, err := os.Stat(wirePath)
			if err != nil {
				continue
			}
			if opts.Metrics != nil {
				opts.Metrics.ConsideredFiles++
			}
			if shouldSkipKimiWirePath(info.ModTime(), opts) {
				if opts.Metrics != nil {
					opts.Metrics.SkippedFiles++
				}
				continue
			}
			paths = append(paths, wirePath)
		}
	}

	if opts.Metrics != nil {
		opts.Metrics.ParsedFiles += len(paths)
	}
	eventBatches := provider.ParseParallel(paths, 0, parseSessionUsageEvents)

	var events []provider.UsageEvent
	for _, batch := range eventBatches {
		events = append(events, batch...)
	}
	if opts.Metrics != nil {
		opts.Metrics.EmittedEvents += len(events)
	}

	return events, nil
}

func shouldSkipKimiWirePath(modTime time.Time, opts provider.UsageEventCollectOptions) bool {
	if !opts.HasRange() {
		return false
	}
	return opts.ShouldSkipFileByModTime(modTime)
}

// discoverSessions walks baseDir/<work-dir>/<session-id>/ and collects sessions
// that have at least one agent wire.jsonl.
func discoverSessions(baseDir string) ([]sessionRef, error) {
	workDirs, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var refs []sessionRef
	for _, wd := range workDirs {
		if !wd.IsDir() {
			continue
		}
		workDirPath := filepath.Join(baseDir, wd.Name())

		sessionDirs, err := os.ReadDir(workDirPath)
		if err != nil {
			continue
		}
		for _, sd := range sessionDirs {
			if !sd.IsDir() {
				continue
			}
			sessionPath := filepath.Join(workDirPath, sd.Name())
			wirePaths := agentWirePaths(sessionPath)
			if len(wirePaths) == 0 {
				continue
			}
			refs = append(refs, sessionRef{
				dir:       sessionPath,
				wirePaths: wirePaths,
			})
		}
	}
	return refs, nil
}

// agentWirePaths returns the wire.jsonl paths of all agents in a session,
// with the main agent first.
func agentWirePaths(sessionPath string) []string {
	agentsDir := filepath.Join(sessionPath, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wirePath := filepath.Join(agentsDir, entry.Name(), "wire.jsonl")
		if _, err := os.Stat(wirePath); err != nil {
			continue
		}
		paths = append(paths, wirePath)
	}
	sort.Slice(paths, func(i, j int) bool {
		iMain := filepath.Base(filepath.Dir(paths[i])) == "main"
		jMain := filepath.Base(filepath.Dir(paths[j])) == "main"
		if iMain != jMain {
			return iMain
		}
		return paths[i] < paths[j]
	})
	return paths
}

// parseSession parses a single session directory, aggregating all agent wire files.
func parseSession(sessionPath string) (provider.SessionInfo, error) {
	info := provider.SessionInfo{
		ProviderName: "kimi",
		SessionID:    filepath.Base(sessionPath),
		WorkDirHash:  filepath.Base(filepath.Dir(sessionPath)),
	}

	if state, err := parseSessionState(filepath.Join(sessionPath, "state.json")); err == nil {
		info.Title = state.Title
	}

	wirePaths := agentWirePaths(sessionPath)
	if len(wirePaths) == 0 {
		return provider.SessionInfo{}, fmt.Errorf("no agent wire.jsonl under %s", sessionPath)
	}

	for _, wirePath := range wirePaths {
		usage, turns, startTime, endTime, modelName, err := parseWireJSONL(wirePath)
		if err != nil {
			return provider.SessionInfo{}, err
		}

		info.TokenUsage.InputOther += usage.InputOther
		info.TokenUsage.Output += usage.Output
		info.TokenUsage.InputCacheRead += usage.InputCacheRead
		info.TokenUsage.InputCacheCreate += usage.InputCacheCreate

		// Only the main agent's prompts count as user-facing turns.
		if filepath.Base(filepath.Dir(wirePath)) == "main" {
			info.Turns += turns
		}
		if !startTime.IsZero() && (info.StartTime.IsZero() || startTime.Before(info.StartTime)) {
			info.StartTime = startTime
		}
		if endTime.After(info.EndTime) {
			info.EndTime = endTime
		}
		if info.ModelName == "" {
			info.ModelName = normalizeKimiModelName(modelName)
		}
	}

	return info, nil
}

// parseSessionUsageEvents parses one agent wire.jsonl. The session directory is
// three levels up: <work-dir>/<session-id>/agents/<agent>/wire.jsonl.
func parseSessionUsageEvents(wirePath string) ([]provider.UsageEvent, error) {
	sessionPath := filepath.Dir(filepath.Dir(filepath.Dir(wirePath)))
	baseEvent := provider.UsageEvent{
		ProviderName: "kimi",
		SessionID:    filepath.Base(sessionPath),
		WorkDirHash:  filepath.Base(filepath.Dir(sessionPath)),
		SourcePath:   wirePath,
	}

	if state, err := parseSessionState(filepath.Join(sessionPath, "state.json")); err == nil {
		baseEvent.Title = state.Title
	}

	events, modelName, err := parseKimiUsageEvents(wirePath, baseEvent)
	if err != nil {
		return nil, err
	}

	resolvedModelName := normalizeKimiModelName(modelName)
	if resolvedModelName != "" {
		for i := range events {
			if events[i].ModelName == "" {
				events[i].ModelName = resolvedModelName
			}
		}
	}

	return events, nil
}

// parseSessionState reads and parses a state.json file.
func parseSessionState(path string) (sessionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionState{}, err
	}

	var state sessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return sessionState{}, err
	}
	return state, nil
}

// parseWireJSONL parses a wire.jsonl file and extracts token usage, turn count, and timestamps.
func parseWireJSONL(path string) (provider.TokenUsage, int, time.Time, time.Time, string, error) {
	var usage provider.TokenUsage
	var turns int
	var startTime, endTime time.Time
	var modelName string

	err := scanWireLines(path, func(line []byte, _ int) {
		var event wireEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines
			return
		}

		if ts := timeFromUnixMillis(event.Time); !ts.IsZero() {
			if startTime.IsZero() || ts.Before(startTime) {
				startTime = ts
			}
			if ts.After(endTime) {
				endTime = ts
			}
		}

		switch event.Type {
		case "usage.record":
			if event.Usage == nil {
				return
			}
			if modelName == "" {
				modelName = event.Model
			}
			usage.InputOther += event.Usage.InputOther
			usage.Output += event.Usage.Output
			usage.InputCacheRead += event.Usage.InputCacheRead
			usage.InputCacheCreate += event.Usage.InputCacheCreation

		case "turn.prompt":
			// Only user prompts count as turns; system triggers (goal
			// continuation, retries) are not user-facing turns.
			if event.Origin == nil || event.Origin.Kind == "" || event.Origin.Kind == "user" {
				turns++
			}
		}
	})
	if err != nil {
		return provider.TokenUsage{}, 0, time.Time{}, time.Time{}, "", err
	}

	return usage, turns, startTime, endTime, modelName, nil
}

func parseKimiUsageEvents(wirePath string, baseEvent provider.UsageEvent) ([]provider.UsageEvent, string, error) {
	var events []provider.UsageEvent
	var modelName string

	err := scanWireLines(wirePath, func(line []byte, lineNo int) {
		var event wireEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return
		}
		if event.Type != "usage.record" {
			return
		}
		if event.Usage == nil {
			return
		}
		if modelName == "" {
			modelName = event.Model
		}

		usageEvent := baseEvent
		if usageEvent.ProviderName == "" {
			usageEvent.ProviderName = "kimi"
		}
		if usageEvent.ModelName == "" {
			usageEvent.ModelName = normalizeKimiModelName(event.Model)
		}
		usageEvent.Timestamp = timeFromUnixMillis(event.Time)
		usageEvent.TokenUsage = provider.TokenUsage{
			InputOther:       event.Usage.InputOther,
			Output:           event.Usage.Output,
			InputCacheRead:   event.Usage.InputCacheRead,
			InputCacheCreate: event.Usage.InputCacheCreation,
		}
		usageEvent.SourcePath = wirePath
		usageEvent.EventID = wirePath + ":" + strconv.Itoa(lineNo)
		events = append(events, usageEvent)
	})
	if err != nil {
		return nil, "", err
	}

	return events, modelName, nil
}

// scanWireLines iterates a wire.jsonl file line by line. It uses bufio.Reader
// instead of bufio.Scanner because wire lines (e.g. profile.bind with a full
// system prompt) can exceed any fixed scanner buffer.
func scanWireLines(path string, handle func(line []byte, lineNo int)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReaderSize(f, 1024*1024)
	lineNo := 0
	for {
		line, err := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			lineNo++
			handle(line, lineNo)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// timeFromUnixMillis converts a Unix timestamp in milliseconds to time.Time.
func timeFromUnixMillis(ts int64) time.Time {
	if ts <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ts)
}

// normalizeKimiModelName strips the builtin profile namespace from Kimi Code
// model aliases (e.g. "kimi-code/k3-256k" -> "k3-256k").
func normalizeKimiModelName(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	modelName = strings.TrimPrefix(modelName, "kimi-code/")
	return modelName
}

func defaultKimiSessionsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kimi-code", "sessions")
}
