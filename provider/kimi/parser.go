package kimi

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/miss-you/codetok/provider"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements provider.Provider for the Kimi CLI.
type Provider struct{}

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
}

// CollectSessions scans baseDir for Kimi session directories and returns session info.
// The expected directory layout is: baseDir/<work-dir-hash>/<session-uuid>/wire.jsonl
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(home, ".kimi", "sessions")
	}

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
		return parseSession(path, pathToHash[path])
	})

	return sessions, nil
}

// parseSession parses a single session directory.
func parseSession(sessionPath, workDirHash string) (provider.SessionInfo, error) {
	info := provider.SessionInfo{
		ProviderName: "kimi",
		WorkDirHash:  workDirHash,
	}

	// Parse metadata.json
	meta, err := parseMetadata(filepath.Join(sessionPath, "metadata.json"))
	if err == nil {
		info.SessionID = meta.SessionID
		info.Title = meta.Title
	} else {
		// Use directory name as session ID if metadata is missing
		info.SessionID = filepath.Base(sessionPath)
	}

	// Parse wire.jsonl
	wirePath := filepath.Join(sessionPath, "wire.jsonl")
	usage, turns, startTime, endTime, err := parseWireJSONL(wirePath)
	if err != nil {
		return provider.SessionInfo{}, err
	}

	info.TokenUsage = usage
	info.Turns = turns
	info.StartTime = startTime
	info.EndTime = endTime

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
func parseWireJSONL(path string) (provider.TokenUsage, int, time.Time, time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return provider.TokenUsage{}, 0, time.Time{}, time.Time{}, err
	}
	defer f.Close()

	var usage provider.TokenUsage
	var turns int
	var startTime, endTime time.Time

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

	return usage, turns, startTime, endTime, scanner.Err()
}

// timeFromUnix converts a Unix timestamp (float64 seconds) to time.Time.
func timeFromUnix(ts float64) time.Time {
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}
