package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/miss-you/codetok/provider"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements provider.Provider for the Codex CLI.
type Provider struct{}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "codex"
}

// codexEvent represents a single line in a Codex rollout JSONL file.
type codexEvent struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// sessionMetaPayload holds the session_meta payload.
type sessionMetaPayload struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Cwd       string `json:"cwd"`
}

// eventMsgPayload holds the event_msg payload envelope.
type eventMsgPayload struct {
	Type    string          `json:"type"`
	Model   string          `json:"model"`
	Message string          `json:"message"`
	Info    json.RawMessage `json:"info"`
}

var codexModelPaths = [][]string{
	{"model"},
	{"model_name"},
	{"modelName"},
	{"model_id"},
	{"modelId"},
	{"selected_model"},
	{"default_model"},
	{"context", "model"},
	{"context", "model_name"},
	{"context", "modelName"},
	{"context", "model_id"},
	{"context", "modelId"},
	{"info", "model"},
	{"info", "model_name"},
	{"info", "modelName"},
	{"info", "model_id"},
	{"info", "modelId"},
	{"payload", "model"},
	{"payload", "model_name"},
	{"payload", "modelName"},
	{"payload", "model_id"},
	{"payload", "modelId"},
}

// tokenCountInfo holds the token_count info field.
type tokenCountInfo struct {
	LastTokenUsage  *codexTokenUsage `json:"last_token_usage"`
	TotalTokenUsage *codexTokenUsage `json:"total_token_usage"`
}

type codexTokenUsage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	TotalTokens           int `json:"total_tokens"`
}

type codexUsageState struct {
	previousTotal  *codexTokenUsage
	pendingLast    codexTokenUsage
	hasPendingLast bool
}

func (u codexTokenUsage) toProviderTokenUsage() provider.TokenUsage {
	return provider.TokenUsage{
		InputOther:     u.InputTokens - u.CachedInputTokens,
		InputCacheRead: u.CachedInputTokens,
		Output:         u.OutputTokens,
	}
}

// CollectSessions scans baseDir for Codex session files and returns session info.
// The expected directory layout is: baseDir/<year>/<month>/<day>/rollout-*.jsonl
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	paths, err := collectCodexSessionPaths(baseDir)
	if err != nil {
		return nil, err
	}

	// Parse all sessions in parallel.
	sessions := provider.ParseParallel(paths, 0, func(path string) (provider.SessionInfo, error) {
		return parseCodexSession(path)
	})

	return sessions, nil
}

func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error) {
	return p.collectUsageEvents(baseDir, provider.UsageEventCollectOptions{})
}

func (p *Provider) CollectUsageEventsInRange(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	return p.collectUsageEvents(baseDir, opts)
}

func (p *Provider) collectUsageEvents(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	paths, err := collectCodexSessionPaths(baseDir)
	if err != nil {
		return nil, err
	}

	paths = filterCodexUsageEventPaths(paths, opts)
	if opts.Metrics != nil {
		opts.Metrics.ParsedFiles += len(paths)
	}
	events := provider.ParseUsageEventsParallel(paths, 0, parseCodexUsageEvents)
	if opts.Metrics != nil {
		opts.Metrics.EmittedEvents += len(events)
	}
	return events, nil
}

func filterCodexUsageEventPaths(paths []string, opts provider.UsageEventCollectOptions) []string {
	if !opts.HasRange() {
		if opts.Metrics != nil {
			opts.Metrics.ConsideredFiles += len(paths)
		}
		return paths
	}
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if opts.Metrics != nil {
			opts.Metrics.ConsideredFiles++
		}
		if codexPathMayOverlapRange(path, opts) {
			filtered = append(filtered, path)
			continue
		}
		info, err := os.Stat(path)
		if err == nil && !opts.ShouldSkipFileByModTime(info.ModTime()) {
			filtered = append(filtered, path)
			continue
		}
		if opts.Metrics != nil {
			opts.Metrics.SkippedFiles++
		}
	}
	return filtered
}

func codexPathMayOverlapRange(path string, opts provider.UsageEventCollectOptions) bool {
	if opts.Since.IsZero() {
		return true
	}
	pathDate, ok := codexPathLocalDate(path, opts.Location)
	if !ok {
		return true
	}
	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}
	sinceLocal := opts.Since.In(loc)
	sinceDate := time.Date(sinceLocal.Year(), sinceLocal.Month(), sinceLocal.Day(), 0, 0, 0, 0, loc)
	return !pathDate.Before(sinceDate.AddDate(0, 0, -1))
}

func codexPathLocalDate(path string, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		loc = time.Local
	}
	day := filepath.Base(filepath.Dir(path))
	month := filepath.Base(filepath.Dir(filepath.Dir(path)))
	year := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
	parsed, err := time.ParseInLocation("2006-01-02", year+"-"+month+"-"+day, loc)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func resolveCodexSessionsDir(baseDir string) (string, error) {
	if baseDir != "" {
		return baseDir, nil
	}
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return filepath.Join(codexHome, "sessions"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "sessions"), nil
}

func collectCodexSessionPaths(baseDir string) ([]string, error) {
	baseDir, err := resolveCodexSessionsDir(baseDir)
	if err != nil {
		return nil, err
	}

	var paths []string

	years, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	for _, y := range years {
		if !y.IsDir() {
			continue
		}
		yearPath := filepath.Join(baseDir, y.Name())

		months, err := os.ReadDir(yearPath)
		if err != nil {
			continue
		}

		for _, m := range months {
			if !m.IsDir() {
				continue
			}
			monthPath := filepath.Join(yearPath, m.Name())

			days, err := os.ReadDir(monthPath)
			if err != nil {
				continue
			}

			for _, d := range days {
				if !d.IsDir() {
					continue
				}
				dayPath := filepath.Join(monthPath, d.Name())

				files, err := os.ReadDir(dayPath)
				if err != nil {
					continue
				}

				for _, f := range files {
					if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
						continue
					}
					paths = append(paths, filepath.Join(dayPath, f.Name()))
				}
			}
		}
	}

	return paths, nil
}

// parseCodexSession parses a single Codex rollout JSONL file.
func parseCodexSession(path string) (provider.SessionInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return provider.SessionInfo{}, err
	}
	defer f.Close()

	info := provider.SessionInfo{
		ProviderName: "codex",
	}

	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var usage provider.TokenUsage
	var hasUsage bool
	var usageState codexUsageState
	var startTime, endTime time.Time
	var turns int

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event codexEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines
			continue
		}

		// Track timestamps for start/end
		if event.Timestamp != "" {
			ts, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err == nil {
				if startTime.IsZero() || ts.Before(startTime) {
					startTime = ts
				}
				if ts.After(endTime) {
					endTime = ts
				}
			}
		}

		switch event.Type {
		case "session_meta":
			var meta sessionMetaPayload
			if err := json.Unmarshal(event.Payload, &meta); err != nil {
				continue
			}
			if info.SessionID == "" {
				info.SessionID = meta.ID
			}
			if meta.Timestamp != "" && startTime.IsZero() {
				ts, err := time.Parse(time.RFC3339Nano, meta.Timestamp)
				if err == nil {
					startTime = ts
				}
			}
			if info.ModelName == "" {
				info.ModelName = extractModelFromRawJSON(event.Payload)
			}

		case "event_msg":
			var msg eventMsgPayload
			if err := json.Unmarshal(event.Payload, &msg); err != nil {
				continue
			}
			if info.ModelName == "" {
				candidate := strings.TrimSpace(msg.Model)
				if isLikelyCodexModelName(candidate) {
					info.ModelName = candidate
				}
				if info.ModelName == "" {
					info.ModelName = extractModelFromRawJSON(msg.Info)
				}
				if info.ModelName == "" {
					info.ModelName = extractModelFromRawJSON(event.Payload)
				}
			}

			switch msg.Type {
			case "user_message":
				turns++
				if info.Title == "" && msg.Message != "" {
					info.Title = msg.Message
				}

			case "token_count":
				if msg.Info == nil || string(msg.Info) == "null" {
					continue
				}
				var tci tokenCountInfo
				if err := json.Unmarshal(msg.Info, &tci); err != nil {
					continue
				}
				delta, ok := codexUsageDelta(tci, &usageState)
				if ok && delta.Total() != 0 {
					addCodexTokenUsage(&usage, delta)
					hasUsage = true
				}
			}

		default:
			if info.ModelName == "" {
				info.ModelName = extractModelFromRawJSON(event.Payload)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return provider.SessionInfo{}, err
	}

	if hasUsage {
		info.TokenUsage = usage
	}
	info.Turns = turns
	info.StartTime = startTime
	info.EndTime = endTime

	return info, nil
}

func parseCodexUsageEvents(path string) ([]provider.UsageEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []provider.UsageEvent
	var sessionID string
	var title string
	var currentModel string
	var usageState codexUsageState
	var lineNumber int

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event codexEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		switch event.Type {
		case "session_meta":
			var meta sessionMetaPayload
			if err := json.Unmarshal(event.Payload, &meta); err != nil {
				continue
			}
			if sessionID == "" && strings.TrimSpace(meta.ID) != "" {
				sessionID = strings.TrimSpace(meta.ID)
				for i := range events {
					events[i].SessionID = sessionID
				}
			}
			if model := extractModelFromRawJSON(event.Payload); model != "" {
				currentModel = model
			}

		case "event_msg":
			var msg eventMsgPayload
			if err := json.Unmarshal(event.Payload, &msg); err != nil {
				continue
			}
			if model := firstCodexModel(msg.Model, extractModelFromRawJSON(msg.Info), extractModelFromRawJSON(event.Payload)); model != "" {
				currentModel = model
			}

			switch msg.Type {
			case "user_message":
				if title == "" && strings.TrimSpace(msg.Message) != "" {
					title = strings.TrimSpace(msg.Message)
					for i := range events {
						if events[i].Title == "" {
							events[i].Title = title
						}
					}
				}

			case "token_count":
				if msg.Info == nil || string(msg.Info) == "null" {
					continue
				}
				ts, err := time.Parse(time.RFC3339Nano, event.Timestamp)
				if err != nil {
					continue
				}
				var tci tokenCountInfo
				if err := json.Unmarshal(msg.Info, &tci); err != nil {
					continue
				}
				usage, ok := codexUsageDelta(tci, &usageState)
				if !ok || usage.Total() == 0 {
					continue
				}
				events = append(events, provider.UsageEvent{
					ProviderName: "codex",
					ModelName:    currentModel,
					SessionID:    sessionID,
					Title:        title,
					Timestamp:    ts,
					TokenUsage:   usage,
					SourcePath:   path,
					EventID:      fmt.Sprintf("%s:%d", path, lineNumber),
				})
			}

		default:
			if model := extractModelFromRawJSON(event.Payload); model != "" {
				currentModel = model
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func codexUsageDelta(info tokenCountInfo, state *codexUsageState) (provider.TokenUsage, bool) {
	if info.LastTokenUsage != nil {
		last := *info.LastTokenUsage
		if info.TotalTokenUsage != nil {
			total := *info.TotalTokenUsage
			state.previousTotal = &total
			state.clearPendingLast()
		} else {
			state.addPendingLast(last)
		}
		return last.toProviderTokenUsage(), true
	}
	if info.TotalTokenUsage == nil {
		return provider.TokenUsage{}, false
	}

	total := *info.TotalTokenUsage
	delta := total
	if state.previousTotal != nil && !codexTotalDecreased(total, *state.previousTotal) {
		delta = subtractCodexRawTokenUsage(total, *state.previousTotal)
	}
	if state.hasPendingLast && canSubtractCodexRawTokenUsage(delta, state.pendingLast) {
		delta = subtractCodexRawTokenUsage(delta, state.pendingLast)
	}
	state.previousTotal = &total
	state.clearPendingLast()
	return delta.toProviderTokenUsage(), true
}

func (s *codexUsageState) addPendingLast(usage codexTokenUsage) {
	s.pendingLast = addCodexRawTokenUsage(s.pendingLast, usage)
	s.hasPendingLast = true
}

func (s *codexUsageState) clearPendingLast() {
	s.pendingLast = codexTokenUsage{}
	s.hasPendingLast = false
}

func addCodexRawTokenUsage(dst, src codexTokenUsage) codexTokenUsage {
	return codexTokenUsage{
		InputTokens:           dst.InputTokens + src.InputTokens,
		CachedInputTokens:     dst.CachedInputTokens + src.CachedInputTokens,
		OutputTokens:          dst.OutputTokens + src.OutputTokens,
		ReasoningOutputTokens: dst.ReasoningOutputTokens + src.ReasoningOutputTokens,
		TotalTokens:           dst.TotalTokens + src.TotalTokens,
	}
}

func subtractCodexRawTokenUsage(current, previous codexTokenUsage) codexTokenUsage {
	return codexTokenUsage{
		InputTokens:           current.InputTokens - previous.InputTokens,
		CachedInputTokens:     current.CachedInputTokens - previous.CachedInputTokens,
		OutputTokens:          current.OutputTokens - previous.OutputTokens,
		ReasoningOutputTokens: current.ReasoningOutputTokens - previous.ReasoningOutputTokens,
		TotalTokens:           current.TotalTokens - previous.TotalTokens,
	}
}

func canSubtractCodexRawTokenUsage(current, previous codexTokenUsage) bool {
	return current.InputTokens >= previous.InputTokens &&
		current.CachedInputTokens >= previous.CachedInputTokens &&
		current.OutputTokens >= previous.OutputTokens &&
		current.ReasoningOutputTokens >= previous.ReasoningOutputTokens &&
		current.TotalTokens >= previous.TotalTokens
}

func codexTotalDecreased(current, previous codexTokenUsage) bool {
	return current.InputTokens < previous.InputTokens ||
		current.CachedInputTokens < previous.CachedInputTokens ||
		current.OutputTokens < previous.OutputTokens ||
		current.TotalTokens < previous.TotalTokens
}

func addCodexTokenUsage(dst *provider.TokenUsage, src provider.TokenUsage) {
	dst.InputOther += src.InputOther
	dst.Output += src.Output
	dst.InputCacheRead += src.InputCacheRead
	dst.InputCacheCreate += src.InputCacheCreate
}

func firstCodexModel(candidates ...string) string {
	for _, candidate := range candidates {
		if isLikelyCodexModelName(candidate) {
			return strings.TrimSpace(candidate)
		}
	}
	return ""
}

func extractModelFromRawJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	for _, path := range codexModelPaths {
		if model := extractStringByPath(v, path); isLikelyCodexModelName(model) {
			return model
		}
	}
	return ""
}

func extractStringByPath(root any, path []string) string {
	current := root
	for _, segment := range path {
		node, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := node[segment]
		if !ok {
			return ""
		}
		current = next
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func isLikelyCodexModelName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	lowerName := strings.ToLower(name)
	switch lowerName {
	case "auto", "default", "none", "null", "n/a", "unknown":
		return false
	}
	if strings.Contains(lowerName, "rate limit") || strings.Contains(lowerName, "rate-limit") {
		return false
	}
	return true
}
