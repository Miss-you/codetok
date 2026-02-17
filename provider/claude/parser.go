package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
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
	Timestamp string    `json:"timestamp"`
	Message   claudeMsg `json:"message"`
}

// claudeMsg represents the message field in a Claude Code event.
type claudeMsg struct {
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

	var sessions []provider.SessionInfo

	// Walk the two-level structure: baseDir/<project-slug>/<session>.jsonl
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
			info, err := parseSession(sessionPath, projectSlug)
			if err != nil {
				continue
			}
			sessions = append(sessions, info)
		}
	}

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
	var usage provider.TokenUsage
	var title string

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
				usage.InputOther += event.Message.Usage.InputTokens
				usage.InputCacheRead += event.Message.Usage.CacheReadInputTokens
				usage.InputCacheCreate += event.Message.Usage.CacheCreationInputTokens
				usage.Output += event.Message.Usage.OutputTokens
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
