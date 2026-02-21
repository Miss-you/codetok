package kimi

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/miss-you/codetok/provider"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements provider.Provider for the Kimi CLI.
type Provider struct{}

var (
	createdSessionLogPattern = regexp.MustCompile(`Created new session:\s*([0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12})`)
	modelLogPattern          = regexp.MustCompile(`model=['"]([^'"]+)['"]`)
)

// Name returns the provider name.
func (p *Provider) Name() string {
	return "kimi"
}

// wireEvent represents a single line in wire.jsonl.
type wireEvent struct {
	Timestamp float64 `json:"timestamp"`
	Message   struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	} `json:"message"`
}

// statusPayload holds the StatusUpdate payload.
type statusPayload struct {
	Model      string `json:"model"`
	ModelName  string `json:"model_name"`
	ModelID    string `json:"model_id"`
	TokenUsage struct {
		InputOther         int `json:"input_other"`
		Output             int `json:"output"`
		InputCacheRead     int `json:"input_cache_read"`
		InputCacheCreation int `json:"input_cache_creation"`
	} `json:"token_usage"`
}

// metadata represents the metadata.json file.
type metadata struct {
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
	Model     string `json:"model"`
	ModelName string `json:"model_name"`
	ModelID   string `json:"model_id"`
}

// CollectSessions scans baseDir for Kimi session directories and returns session info.
// The expected directory layout is: baseDir/<work-dir-hash>/<session-uuid>/wire.jsonl
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	if baseDir == "" {
		baseDir = defaultKimiSessionsDir()
	}
	sessionModelIndex := loadSessionModelsFromLogs(detectKimiLogsDir(baseDir))

	// Phase 1: Walk directories, collect all session paths (sequential, fast)
	var paths []string
	pathToHash := make(map[string]string)

	workDirs, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	for _, wd := range workDirs {
		if !wd.IsDir() {
			continue
		}
		workDirHash := wd.Name()
		workDirPath := filepath.Join(baseDir, workDirHash)

		sessionDirs, err := os.ReadDir(workDirPath)
		if err != nil {
			continue
		}

		for _, sd := range sessionDirs {
			if !sd.IsDir() {
				continue
			}
			sessionPath := filepath.Join(workDirPath, sd.Name())
			wirePath := filepath.Join(sessionPath, "wire.jsonl")

			// Skip sessions without wire.jsonl
			if _, err := os.Stat(wirePath); err != nil {
				continue
			}

			paths = append(paths, sessionPath)
			pathToHash[sessionPath] = workDirHash
		}
	}

	// Phase 2: Parse all sessions in parallel
	sessions := provider.ParseParallel(paths, 0, func(path string) (provider.SessionInfo, error) {
		return parseSession(path, pathToHash[path], sessionModelIndex)
	})

	return sessions, nil
}

// parseSession parses a single session directory.
func parseSession(sessionPath, workDirHash string, sessionModelIndex map[string]string) (provider.SessionInfo, error) {
	info := provider.SessionInfo{
		ProviderName: "kimi",
		WorkDirHash:  workDirHash,
	}

	// Parse metadata.json
	meta, err := parseMetadata(filepath.Join(sessionPath, "metadata.json"))
	if err == nil {
		info.SessionID = meta.SessionID
		info.Title = meta.Title
		info.ModelName = normalizeKimiModelName(firstNonEmpty(meta.ModelName, meta.Model, meta.ModelID))
	} else {
		// Use directory name as session ID if metadata is missing
		info.SessionID = filepath.Base(sessionPath)
	}

	// Parse wire.jsonl
	wirePath := filepath.Join(sessionPath, "wire.jsonl")
	usage, turns, startTime, endTime, modelName, err := parseWireJSONL(wirePath)
	if err != nil {
		return provider.SessionInfo{}, err
	}

	info.TokenUsage = usage
	info.Turns = turns
	info.StartTime = startTime
	info.EndTime = endTime
	if info.ModelName == "" {
		info.ModelName = normalizeKimiModelName(modelName)
	}
	if info.ModelName == "" {
		info.ModelName = modelNameFromLogFallback(info.SessionID, sessionPath, sessionModelIndex)
	}

	return info, nil
}

// parseMetadata reads and parses a metadata.json file.
func parseMetadata(path string) (metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return metadata{}, err
	}

	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return metadata{}, err
	}
	return meta, nil
}

// parseWireJSONL parses a wire.jsonl file and extracts token usage, turn count, and timestamps.
func parseWireJSONL(path string) (provider.TokenUsage, int, time.Time, time.Time, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return provider.TokenUsage{}, 0, time.Time{}, time.Time{}, "", err
	}
	defer f.Close()

	var usage provider.TokenUsage
	var turns int
	var startTime, endTime time.Time
	var modelName string

	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event wireEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines
			continue
		}

		switch event.Message.Type {
		case "StatusUpdate":
			var payload statusPayload
			if err := json.Unmarshal(event.Message.Payload, &payload); err != nil {
				continue
			}
			if modelName == "" {
				modelName = firstNonEmpty(payload.ModelName, payload.Model, payload.ModelID)
			}
			usage.InputOther += payload.TokenUsage.InputOther
			usage.Output += payload.TokenUsage.Output
			usage.InputCacheRead += payload.TokenUsage.InputCacheRead
			usage.InputCacheCreate += payload.TokenUsage.InputCacheCreation

		case "TurnBegin":
			turns++
			ts := timeFromUnix(event.Timestamp)
			if startTime.IsZero() || ts.Before(startTime) {
				startTime = ts
			}

		case "TurnEnd":
			ts := timeFromUnix(event.Timestamp)
			if ts.After(endTime) {
				endTime = ts
			}
		}
	}

	return usage, turns, startTime, endTime, modelName, scanner.Err()
}

// timeFromUnix converts a Unix timestamp (float64 seconds) to time.Time.
func timeFromUnix(ts float64) time.Time {
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeKimiModelName(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ""
	}

	alias := strings.ToLower(modelName)
	alias = strings.ReplaceAll(alias, "_", "-")
	alias = strings.ReplaceAll(alias, " ", "-")
	for strings.Contains(alias, "--") {
		alias = strings.ReplaceAll(alias, "--", "-")
	}

	switch alias {
	case "k2.5", "k2-5", "kimi-k2.5", "kimi-k2-5":
		return "kimi-k2.5"
	case "k2-thinking", "k2thinking", "kimi-k2-thinking", "kimi-k2thinking":
		return "kimi-k2-thinking"
	default:
		return modelName
	}
}

func defaultKimiLogsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kimi", "logs")
}

func defaultKimiSessionsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kimi", "sessions")
}

func detectKimiLogsDir(baseDir string) string {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return ""
	}

	// Prefer sibling logs for the provided sessions directory.
	siblingLogsDir := filepath.Join(filepath.Dir(baseDir), "logs")
	if filepath.Base(baseDir) == "sessions" && isDir(siblingLogsDir) {
		return siblingLogsDir
	}

	// For default sessions dir, fall back to default logs dir.
	if filepath.Clean(baseDir) == filepath.Clean(defaultKimiSessionsDir()) {
		if logsDir := defaultKimiLogsDir(); isDir(logsDir) {
			return logsDir
		}
	}
	return ""
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func loadSessionModelsFromLogs(logDir string) map[string]string {
	logDir = strings.TrimSpace(logDir)
	if logDir == "" {
		return nil
	}

	logPaths, err := filepath.Glob(filepath.Join(logDir, "kimi*.log"))
	if err != nil || len(logPaths) == 0 {
		return nil
	}
	sort.Strings(logPaths)

	sessionModelIndex := make(map[string]string)
	for _, logPath := range logPaths {
		mergeSessionModelsFromLog(logPath, sessionModelIndex)
	}

	if len(sessionModelIndex) == 0 {
		return nil
	}
	return sessionModelIndex
}

func mergeSessionModelsFromLog(logPath string, sessionModelIndex map[string]string) {
	f, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentSessionID string
	for scanner.Scan() {
		line := scanner.Text()

		if sessionID := createdSessionIDFromLogLine(line); sessionID != "" {
			currentSessionID = sessionID
			continue
		}
		if currentSessionID == "" || !strings.Contains(line, "Using LLM model:") {
			continue
		}

		modelName := modelNameFromLogLine(line)
		if modelName == "" {
			continue
		}

		if _, exists := sessionModelIndex[currentSessionID]; !exists {
			sessionModelIndex[currentSessionID] = normalizeKimiModelName(modelName)
		}
	}
}

func createdSessionIDFromLogLine(line string) string {
	if !strings.Contains(line, "Created new session:") {
		return ""
	}
	matches := createdSessionLogPattern.FindStringSubmatch(line)
	if len(matches) < 2 {
		return ""
	}
	return normalizeSessionIDForLookup(matches[1])
}

func modelNameFromLogLine(line string) string {
	matches := modelLogPattern.FindStringSubmatch(line)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func modelNameFromLogFallback(sessionID, sessionPath string, sessionModelIndex map[string]string) string {
	if len(sessionModelIndex) == 0 {
		return ""
	}

	candidates := []string{sessionID, filepath.Base(sessionPath)}
	for _, candidate := range candidates {
		lookupID := normalizeSessionIDForLookup(candidate)
		if lookupID == "" {
			continue
		}
		if modelName, exists := sessionModelIndex[lookupID]; exists {
			return modelName
		}
	}
	return ""
}

func normalizeSessionIDForLookup(sessionID string) string {
	return strings.ToLower(strings.TrimSpace(sessionID))
}
