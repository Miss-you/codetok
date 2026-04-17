package codex

import (
	"bufio"
	"bytes"
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
	codexModelFields

	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Cwd       string `json:"cwd"`
}

// eventMsgPayload holds the event_msg payload envelope.
type eventMsgPayload struct {
	codexNamedModelFields

	Type    string          `json:"type"`
	Model   string          `json:"model"`
	Message string          `json:"message"`
	Info    json.RawMessage `json:"info"`
	Context json.RawMessage `json:"context"`
	Payload json.RawMessage `json:"payload"`
}

type codexDirectModelFields struct {
	ModelDirect string `json:"model"`
	codexNamedModelFields
}

type codexNamedModelFields struct {
	ModelName     string `json:"model_name"`
	ModelNameJSON string `json:"modelName"`
	ModelID       string `json:"model_id"`
	ModelIDJSON   string `json:"modelId"`
	SelectedModel string `json:"selected_model"`
	DefaultModel  string `json:"default_model"`
}

type codexModelFields struct {
	codexDirectModelFields

	Context json.RawMessage `json:"context"`
	Info    json.RawMessage `json:"info"`
	Payload json.RawMessage `json:"payload"`
}

// tokenCountInfo holds the token_count info field.
type tokenCountInfo struct {
	codexModelFields

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
				info.ModelName = meta.firstModel()
			}

		case "event_msg":
			var msg eventMsgPayload
			if err := json.Unmarshal(event.Payload, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "user_message":
				if info.ModelName == "" {
					info.ModelName = msg.firstModel(extractModelFromRawJSON(msg.Info))
				}
				turns++
				if info.Title == "" && msg.Message != "" {
					info.Title = msg.Message
				}

			case "token_count":
				if len(msg.Info) == 0 || bytes.Equal(bytes.TrimSpace(msg.Info), []byte("null")) {
					if info.ModelName == "" {
						info.ModelName = msg.firstModel("")
					}
					continue
				}
				var tci tokenCountInfo
				if err := json.Unmarshal(msg.Info, &tci); err != nil {
					if info.ModelName == "" {
						info.ModelName = msg.firstModel(extractModelFromRawJSON(msg.Info))
					}
					continue
				}
				if info.ModelName == "" {
					info.ModelName = msg.firstModel(tci.firstModel())
				}
				delta, ok := codexUsageDelta(tci, &usageState)
				if ok && delta.Total() != 0 {
					addCodexTokenUsage(&usage, delta)
					hasUsage = true
				}
			default:
				if info.ModelName == "" {
					info.ModelName = msg.firstModel(extractModelFromRawJSON(msg.Info))
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
			if model := meta.firstModel(); model != "" {
				currentModel = model
			}

		case "event_msg":
			var msg eventMsgPayload
			if err := json.Unmarshal(event.Payload, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "user_message":
				if model := msg.firstModel(extractModelFromRawJSON(msg.Info)); model != "" {
					currentModel = model
				}
				if title == "" && strings.TrimSpace(msg.Message) != "" {
					title = strings.TrimSpace(msg.Message)
					for i := range events {
						if events[i].Title == "" {
							events[i].Title = title
						}
					}
				}

			case "token_count":
				if len(msg.Info) == 0 || bytes.Equal(bytes.TrimSpace(msg.Info), []byte("null")) {
					if model := msg.firstModel(""); model != "" {
						currentModel = model
					}
					continue
				}
				var tci tokenCountInfo
				if err := json.Unmarshal(msg.Info, &tci); err != nil {
					if model := msg.firstModel(extractModelFromRawJSON(msg.Info)); model != "" {
						currentModel = model
					}
					continue
				}
				if model := msg.firstModel(tci.firstModel()); model != "" {
					currentModel = model
				}
				ts, err := time.Parse(time.RFC3339Nano, event.Timestamp)
				if err != nil {
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

			default:
				if model := msg.firstModel(extractModelFromRawJSON(msg.Info)); model != "" {
					currentModel = model
				}
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

func (f codexDirectModelFields) firstModel() string {
	return firstCodexModel(
		f.ModelDirect,
		f.codexNamedModelFields.firstModel(),
	)
}

func (f codexNamedModelFields) firstModel() string {
	return firstCodexModel(
		f.ModelName,
		f.ModelNameJSON,
		f.ModelID,
		f.ModelIDJSON,
		f.SelectedModel,
		f.DefaultModel,
	)
}

func (f codexModelFields) firstModel() string {
	if model := f.codexDirectModelFields.firstModel(); model != "" {
		return model
	}
	return firstCodexModel(
		extractModelFromRawJSON(f.Context),
		extractModelFromRawJSON(f.Info),
		extractModelFromRawJSON(f.Payload),
	)
}

func (m eventMsgPayload) firstModel(infoModel string) string {
	return firstCodexModel(
		m.Model,
		infoModel,
		m.payloadModelWithoutInfo(),
	)
}

func (m eventMsgPayload) payloadModelWithoutInfo() string {
	return firstCodexModel(
		m.codexNamedModelFields.firstModel(),
		extractModelFromRawJSON(m.Context),
		extractModelFromRawJSON(m.Payload),
	)
}

const maxCodexModelExtractionDepth = 1

func extractModelFromRawJSON(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || raw[0] != '{' {
		return ""
	}

	model, _ := extractModelFromObject(raw, 0)
	return model
}

func extractModelFromObject(raw []byte, depth int) (string, bool) {
	var direct codexDirectModelFields
	var nested codexNestedModelRaw

	ok := scanJSONObjectFields(raw, func(key, value []byte) bool {
		if handleEscapedCodexModelKey(key, value, &direct, &nested) {
			return true
		}
		switch {
		case bytes.Equal(key, []byte("model")):
			direct.ModelDirect = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("model_name")):
			direct.ModelName = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("modelName")):
			direct.ModelNameJSON = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("model_id")):
			direct.ModelID = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("modelId")):
			direct.ModelIDJSON = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("selected_model")):
			direct.SelectedModel = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("default_model")):
			direct.DefaultModel = modelStringFromJSONValue(value)
		case bytes.Equal(key, []byte("context")):
			nested.contextRaw, nested.contextSet = value, true
		case bytes.Equal(key, []byte("info")):
			nested.infoRaw, nested.infoSet = value, true
		case bytes.Equal(key, []byte("payload")):
			nested.payloadRaw, nested.payloadSet = value, true
		}
		return true
	})
	if !ok {
		return "", false
	}
	if model := direct.firstModel(); model != "" {
		return model, true
	}
	if depth >= maxCodexModelExtractionDepth {
		return "", true
	}
	for _, candidate := range []struct {
		raw []byte
		set bool
	}{
		{nested.contextRaw, nested.contextSet},
		{nested.infoRaw, nested.infoSet},
		{nested.payloadRaw, nested.payloadSet},
	} {
		if !candidate.set {
			continue
		}
		model, ok := extractModelFromObject(bytes.TrimSpace(candidate.raw), depth+1)
		if ok && model != "" {
			return model, true
		}
	}
	return "", true
}

type codexNestedModelRaw struct {
	contextRaw []byte
	contextSet bool
	infoRaw    []byte
	infoSet    bool
	payloadRaw []byte
	payloadSet bool
}

func handleEscapedCodexModelKey(key, value []byte, direct *codexDirectModelFields, nested *codexNestedModelRaw) bool {
	if !bytes.Contains(key, []byte("\\")) {
		return false
	}
	keyJSON := make([]byte, 0, len(key)+2)
	keyJSON = append(keyJSON, '"')
	keyJSON = append(keyJSON, key...)
	keyJSON = append(keyJSON, '"')
	name, ok := unquoteJSONString(keyJSON)
	if !ok {
		return true
	}
	switch name {
	case "model":
		direct.ModelDirect = modelStringFromJSONValue(value)
	case "model_name":
		direct.ModelName = modelStringFromJSONValue(value)
	case "modelName":
		direct.ModelNameJSON = modelStringFromJSONValue(value)
	case "model_id":
		direct.ModelID = modelStringFromJSONValue(value)
	case "modelId":
		direct.ModelIDJSON = modelStringFromJSONValue(value)
	case "selected_model":
		direct.SelectedModel = modelStringFromJSONValue(value)
	case "default_model":
		direct.DefaultModel = modelStringFromJSONValue(value)
	case "context":
		nested.contextRaw, nested.contextSet = value, true
	case "info":
		nested.infoRaw, nested.infoSet = value, true
	case "payload":
		nested.payloadRaw, nested.payloadSet = value, true
	}
	return true
}

func scanJSONObjectFields(raw []byte, visit func(key, value []byte) bool) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return false
	}

	for i := 1; i < len(raw); {
		i = skipJSONSpace(raw, i)
		if i >= len(raw) {
			return false
		}
		if raw[i] == '}' {
			return true
		}
		if raw[i] == ',' {
			i++
			continue
		}
		if raw[i] != '"' {
			return false
		}
		keyStart := i + 1
		keyEnd, next, ok := scanJSONString(raw, i)
		if !ok {
			return false
		}
		key := raw[keyStart:keyEnd]
		i = skipJSONSpace(raw, next)
		if i >= len(raw) || raw[i] != ':' {
			return false
		}
		i = skipJSONSpace(raw, i+1)
		valueStart := i
		valueEnd, ok := scanJSONValue(raw, i)
		if !ok {
			return false
		}
		if !visit(key, raw[valueStart:valueEnd]) {
			return true
		}
		i = valueEnd
	}
	return false
}

func modelStringFromJSONValue(value []byte) string {
	value = bytes.TrimSpace(value)
	if len(value) == 0 || value[0] != '"' {
		return ""
	}
	end, next, ok := scanJSONString(value, 0)
	if !ok || next != len(value) {
		return ""
	}
	rawString := value[:end+1]
	var model string
	if bytes.Contains(rawString, []byte("\\")) {
		unquoted, ok := unquoteJSONString(rawString)
		if !ok {
			return ""
		}
		model = unquoted
	} else {
		model = string(value[1:end])
	}
	return firstCodexModel(model)
}

func unquoteJSONString(raw []byte) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func scanJSONValue(raw []byte, start int) (int, bool) {
	if start >= len(raw) {
		return 0, false
	}
	switch raw[start] {
	case '"':
		_, next, ok := scanJSONString(raw, start)
		return next, ok
	case '{', '[':
		return scanJSONComposite(raw, start)
	default:
		i := start
		for i < len(raw) && raw[i] != ',' && raw[i] != '}' && raw[i] != ']' {
			i++
		}
		return i, true
	}
}

func scanJSONComposite(raw []byte, start int) (int, bool) {
	depth := 0
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			_, next, ok := scanJSONString(raw, i)
			if !ok {
				return 0, false
			}
			i = next - 1
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1, true
			}
			if depth < 0 {
				return 0, false
			}
		}
	}
	return 0, false
}

func scanJSONString(raw []byte, start int) (endQuote int, next int, ok bool) {
	for i := start + 1; i < len(raw); i++ {
		switch raw[i] {
		case '\\':
			i++
		case '"':
			return i, i + 1, true
		}
	}
	return 0, 0, false
}

func skipJSONSpace(raw []byte, i int) int {
	for i < len(raw) {
		switch raw[i] {
		case ' ', '\n', '\r', '\t':
			i++
		default:
			return i
		}
	}
	return i
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
