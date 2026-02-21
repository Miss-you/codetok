package provider

import "time"

// TokenUsage holds token counts from a session or aggregation.
type TokenUsage struct {
	InputOther       int `json:"input_other"`
	Output           int `json:"output"`
	InputCacheRead   int `json:"input_cache_read"`
	InputCacheCreate int `json:"input_cache_creation"`
}

// TotalInput returns the sum of all input token fields.
func (t TokenUsage) TotalInput() int {
	return t.InputOther + t.InputCacheRead + t.InputCacheCreate
}

// Total returns the sum of all token fields.
func (t TokenUsage) Total() int {
	return t.TotalInput() + t.Output
}

// SessionInfo represents a single coding session with aggregated token usage.
type SessionInfo struct {
	ProviderName string
	ModelName    string
	SessionID    string
	Title        string
	WorkDirHash  string
	StartTime    time.Time
	EndTime      time.Time
	Turns        int
	TokenUsage   TokenUsage
}

// DailyStats represents aggregated token usage for a single day.
type DailyStats struct {
	Date string `json:"date"` // "2006-01-02"
	// ProviderName always represents the CLI/provider dimension.
	// It may be empty when a non-provider grouping spans multiple providers.
	ProviderName string     `json:"provider"`
	GroupBy      string     `json:"group_by"`
	Group        string     `json:"group"`
	Providers    []string   `json:"providers,omitempty"`
	Sessions     int        `json:"sessions"`
	TokenUsage   TokenUsage `json:"token_usage"`
}

// Provider defines the interface for collecting session data from a CLI tool.
type Provider interface {
	Name() string
	CollectSessions(baseDir string) ([]SessionInfo, error)
}
