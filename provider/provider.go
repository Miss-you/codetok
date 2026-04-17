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

// UsageEvent represents a timestamped token usage delta from a provider log.
type UsageEvent struct {
	ProviderName string
	ModelName    string
	SessionID    string
	Title        string
	WorkDirHash  string
	Timestamp    time.Time
	TokenUsage   TokenUsage
	SourcePath   string
	EventID      string
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

// UsageEventProvider is implemented by providers that can emit native usage events.
type UsageEventProvider interface {
	Provider
	CollectUsageEvents(baseDir string) ([]UsageEvent, error)
}

// UsageEventCollectOptions carries an optional date window for candidate usage-event collection.
// Since and Until should be localized day bounds. Providers may use these values
// to reduce candidates, but final command attribution still belongs to stats.
type UsageEventCollectOptions struct {
	Since    time.Time
	Until    time.Time
	Location *time.Location
	Metrics  *UsageEventCollectMetrics
}

// UsageEventCollectMetrics records candidate filtering work for tests and benchmarks.
type UsageEventCollectMetrics struct {
	ConsideredFiles int
	SkippedFiles    int
	// ParsedFiles counts candidate files handed to a parser. Parsers may still
	// discard malformed local files according to the provider's existing rules.
	ParsedFiles   int
	EmittedEvents int
}

// RangeAwareUsageEventProvider is implemented by providers that can narrow usage-event collection.
type RangeAwareUsageEventProvider interface {
	UsageEventProvider
	CollectUsageEventsInRange(baseDir string, opts UsageEventCollectOptions) ([]UsageEvent, error)
}

// HasRange reports whether collection has any date bound.
func (o UsageEventCollectOptions) HasRange() bool {
	return !o.Since.IsZero() || !o.Until.IsZero()
}

// ContainsTimestamp mirrors stats.FilterEventsByDateRange's localized inclusive date-key semantics.
func (o UsageEventCollectOptions) ContainsTimestamp(ts time.Time) bool {
	if !o.HasRange() {
		return true
	}
	loc := o.Location
	if loc == nil {
		loc = time.Local
	}
	date := ts.In(loc).Format("2006-01-02")
	if !o.Since.IsZero() && date < o.Since.In(loc).Format("2006-01-02") {
		return false
	}
	if !o.Until.IsZero() && date > o.Until.In(loc).Format("2006-01-02") {
		return false
	}
	return true
}

// ShouldSkipFileByModTime returns true only for files safely inactive before Since.
func (o UsageEventCollectOptions) ShouldSkipFileByModTime(modTime time.Time) bool {
	return !o.Since.IsZero() && modTime.Before(o.Since)
}
