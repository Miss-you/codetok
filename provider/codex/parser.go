package codex

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

// tokenCountInfo holds the token_count info field.
type tokenCountInfo struct {
	TotalTokenUsage struct {
		InputTokens       int `json:"input_tokens"`
		CachedInputTokens int `json:"cached_input_tokens"`
		OutputTokens      int `json:"output_tokens"`
		TotalTokens       int `json:"total_tokens"`
	} `json:"total_token_usage"`
}

// CollectSessions scans baseDir for Codex session files and returns session info.
// The expected directory layout is: baseDir/<year>/<month>/<day>/rollout-*.jsonl
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(home, ".codex", "sessions")
	}

	// Phase 1: Walk directories, collect all session file paths (sequential, fast)
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

	// Phase 2: Parse all sessions in parallel
	sessions := provider.ParseParallel(paths, 0, func(path string) (provider.SessionInfo, error) {
		return parseCodexSession(path)
	})

	return sessions, nil
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

	var lastTokenUsage *provider.TokenUsage
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
			info.SessionID = meta.ID
			if meta.Timestamp != "" {
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
				info.ModelName = strings.TrimSpace(msg.Model)
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
				tu := tci.TotalTokenUsage
				// Cumulative: take the latest value (overwrite)
				usage := provider.TokenUsage{
					InputOther:     tu.InputTokens - tu.CachedInputTokens,
					InputCacheRead: tu.CachedInputTokens,
					Output:         tu.OutputTokens,
					// Codex doesn't report InputCacheCreate
				}
				lastTokenUsage = &usage
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

	if lastTokenUsage != nil {
		info.TokenUsage = *lastTokenUsage
	}
	info.Turns = turns
	info.StartTime = startTime
	info.EndTime = endTime

	return info, nil
}

func extractModelFromRawJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return extractModelFromAny(v)
}

func extractModelFromAny(v any) string {
	switch x := v.(type) {
	case map[string]any:
		// Prefer common model key names first.
		for _, key := range []string{
			"model",
			"model_name",
			"modelName",
			"model_id",
			"modelId",
			"selected_model",
			"default_model",
			"limit_name",
		} {
			if s, ok := x[key].(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}

		// Then recurse into nested fields.
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if model := extractModelFromAny(x[k]); model != "" {
				return model
			}
		}
	case []any:
		for _, item := range x {
			if model := extractModelFromAny(item); model != "" {
				return model
			}
		}
	}
	return ""
}
