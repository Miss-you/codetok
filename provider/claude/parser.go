package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Miss-you/codetok/provider"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements provider.Provider for Claude Code.
type Provider struct{}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "claude"
}

// claudeEvent represents a single line in a Claude Code JSONL session file.
type claudeEvent struct {
	Type      string    `json:"type"`
	UserType  string    `json:"userType"`
	SessionID string    `json:"sessionId"`
	RequestID string    `json:"requestId"`
	Timestamp string    `json:"timestamp"`
	Message   claudeMsg `json:"message"`
}

// claudeMsg represents the message field in a Claude Code event.
type claudeMsg struct {
	ID      string          `json:"id"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Usage   *claudeUsage    `json:"usage"`
}

// claudeUsage represents the usage field in an assistant message.
type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// CollectSessions scans baseDir for Claude Code session files and returns session info.
// The expected layout is: baseDir/<project-slug>/<session-uuid>.jsonl
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(home, ".claude", "projects")
	}

	// Phase 1: Walk directories, collect all session file paths (sequential, fast)
	var paths []string
	pathToSlug := make(map[string]string)

	projectDirs, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	for _, pd := range projectDirs {
		if !pd.IsDir() {
			continue
		}
		projectSlug := pd.Name()
		projectPath := filepath.Join(baseDir, projectSlug)

		entries, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}

			sessionPath := filepath.Join(projectPath, entry.Name())
			paths = append(paths, sessionPath)
			pathToSlug[sessionPath] = projectSlug
		}
	}

	// Phase 2: Parse all sessions in parallel
	sessions := provider.ParseParallel(paths, 0, func(path string) (provider.SessionInfo, error) {
		return parseSession(path, pathToSlug[path])
	})

	return sessions, nil
}

// parseSession parses a single Claude Code JSONL session file.
func parseSession(path, projectSlug string) (provider.SessionInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return provider.SessionInfo{}, err
	}
	defer f.Close()

	info := provider.SessionInfo{
		ProviderName: "claude",
		WorkDirHash:  projectSlug,
	}

	var turns int
	var startTime, endTime time.Time
	var title string

	// dedupUsage maps "messageId:requestId" to the latest usage seen for that key.
	// Streaming causes the same assistant message to appear multiple times with
	// increasing token counts; we keep only the last (final) entry per key.
	type usageEntry struct {
		inputOther       int
		inputCacheRead   int
		inputCacheCreate int
		output           int
	}
	dedupUsage := make(map[string]usageEntry)
	var uniqueCounter int // fallback counter for entries with no dedup key

	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event claudeEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, event.Timestamp)

		if !ts.IsZero() {
			if startTime.IsZero() || ts.Before(startTime) {
				startTime = ts
			}
			if ts.After(endTime) {
				endTime = ts
			}
		}

		switch event.Type {
		case "user":
			if event.UserType != "" && event.UserType != "external" {
				continue
			}
			turns++
			// Use the first user message as the title
			if title == "" {
				title = extractUserText(event.Message.Content)
			}

		case "assistant":
			if info.SessionID == "" {
				info.SessionID = event.SessionID
			}
			if event.Message.Usage != nil {
				key := dedupKey(event.Message.ID, event.RequestID, &uniqueCounter)
				dedupUsage[key] = usageEntry{
					inputOther:       event.Message.Usage.InputTokens,
					inputCacheRead:   event.Message.Usage.CacheReadInputTokens,
					inputCacheCreate: event.Message.Usage.CacheCreationInputTokens,
					output:           event.Message.Usage.OutputTokens,
				}
			}
		}

		// Capture sessionId from any event
		if info.SessionID == "" && event.SessionID != "" {
			info.SessionID = event.SessionID
		}
	}

	if err := scanner.Err(); err != nil {
		return provider.SessionInfo{}, err
	}

	// Sum deduplicated usage entries
	var usage provider.TokenUsage
	for _, u := range dedupUsage {
		usage.InputOther += u.inputOther
		usage.InputCacheRead += u.inputCacheRead
		usage.InputCacheCreate += u.inputCacheCreate
		usage.Output += u.output
	}

	// Use filename (without extension) as session ID fallback
	if info.SessionID == "" {
		base := filepath.Base(path)
		info.SessionID = strings.TrimSuffix(base, ".jsonl")
	}

	info.Title = truncateTitle(title, 80)
	info.Turns = turns
	info.TokenUsage = usage
	info.StartTime = startTime
	info.EndTime = endTime

	return info, nil
}

// dedupKey builds a deduplication key from messageId and requestId.
// If both are empty, it returns a unique key so the entry is never merged.
func dedupKey(messageID, requestID string, counter *int) string {
	if messageID == "" && requestID == "" {
		*counter++
		return "_unique_" + strconv.Itoa(*counter)
	}
	return messageID + ":" + requestID
}

// extractUserText extracts text from a user message content field.
// Content can be a plain string or an array of content blocks.
func extractUserText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}

	return ""
}

// truncateTitle truncates a string to max runes.
func truncateTitle(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
