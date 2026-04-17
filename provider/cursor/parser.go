package cursor

import (
	"encoding/csv"
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

// Provider implements provider.Provider for Cursor CSV exports.
type Provider struct{}

const (
	cursorDateHeader             = "Date"
	cursorKindHeader             = "Kind"
	cursorModelHeader            = "Model"
	cursorInputCacheCreateHeader = "Input (w/ Cache Write)"
	cursorInputOtherHeader       = "Input (w/o Cache Write)"
	cursorInputCacheReadHeader   = "Cache Read"
	cursorOutputHeader           = "Output Tokens"
)

var cursorRequiredHeaders = []string{
	cursorDateHeader,
	cursorModelHeader,
	cursorInputCacheCreateHeader,
	cursorInputOtherHeader,
	cursorInputCacheReadHeader,
	cursorOutputHeader,
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "cursor"
}

// CollectSessions scans baseDir for Cursor usage export CSV files and returns one
// session-like record per CSV row.
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	paths, err := resolveCursorCSVPaths(baseDir)
	if err != nil {
		return nil, err
	}

	var sessions []provider.SessionInfo
	for _, path := range paths {
		parsed, err := parseUsageCSV(path)
		if err != nil {
			continue
		}
		sessions = append(sessions, parsed...)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if !sessions[i].StartTime.Equal(sessions[j].StartTime) {
			return sessions[i].StartTime.Before(sessions[j].StartTime)
		}
		return sessions[i].SessionID < sessions[j].SessionID
	})

	return sessions, nil
}

// CollectUsageEvents scans Cursor CSV exports and returns one timestamped usage
// event per valid CSV row.
func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error) {
	return p.collectUsageEvents(baseDir, provider.UsageEventCollectOptions{})
}

func (p *Provider) CollectUsageEventsInRange(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	return p.collectUsageEvents(baseDir, opts)
}

func (p *Provider) collectUsageEvents(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	paths, err := resolveCursorCSVPaths(baseDir)
	if err != nil {
		return nil, err
	}

	var events []provider.UsageEvent
	for _, path := range paths {
		if opts.Metrics != nil {
			opts.Metrics.ConsideredFiles++
		}
		parsed, err := parseUsageCSV(path)
		if err != nil {
			continue
		}
		if opts.Metrics != nil {
			opts.Metrics.ParsedFiles++
		}
		for _, session := range parsed {
			event := sessionUsageEvent(path, session)
			if !opts.ContainsTimestamp(event.Timestamp) {
				continue
			}
			events = append(events, event)
		}
	}
	if opts.Metrics != nil {
		opts.Metrics.EmittedEvents += len(events)
	}

	sort.Slice(events, func(i, j int) bool {
		if !events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].Timestamp.Before(events[j].Timestamp)
		}
		return events[i].SessionID < events[j].SessionID
	})

	return events, nil
}

func sessionUsageEvent(sourcePath string, session provider.SessionInfo) provider.UsageEvent {
	return provider.UsageEvent{
		ProviderName: session.ProviderName,
		ModelName:    session.ModelName,
		SessionID:    session.SessionID,
		Title:        session.Title,
		WorkDirHash:  session.WorkDirHash,
		Timestamp:    session.StartTime,
		TokenUsage:   session.TokenUsage,
		SourcePath:   sourcePath,
		EventID:      session.SessionID,
	}
}

func defaultCursorDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codetok", "cursor"), nil
}

func resolveCursorCSVPaths(baseDir string) ([]string, error) {
	if baseDir != "" {
		return collectCSVPathsRecursive(baseDir)
	}

	root, err := defaultCursorDir()
	if err != nil {
		return nil, err
	}

	return collectDefaultCursorCSVPaths(root)
}

func collectDefaultCursorCSVPaths(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".csv") {
			paths = append(paths, filepath.Join(root, entry.Name()))
		}
	}

	for _, subdir := range []string{"imports", "synced"} {
		nested, err := collectCSVPathsRecursiveIfExists(filepath.Join(root, subdir))
		if err != nil {
			return nil, err
		}
		paths = append(paths, nested...)
	}

	sort.Strings(paths)
	return paths, nil
}

func collectCSVPathsRecursiveIfExists(baseDir string) ([]string, error) {
	paths, err := collectCSVPathsRecursive(baseDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return paths, err
}

func collectCSVPathsRecursive(baseDir string) ([]string, error) {
	var paths []string
	if err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".csv") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func parseUsageCSV(path string) ([]provider.SessionInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1

	headerRecord, err := reader.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	header := headerIndex(headerRecord)
	for _, name := range cursorRequiredHeaders {
		if _, ok := header[name]; !ok {
			return nil, os.ErrInvalid
		}
	}

	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var sessions []provider.SessionInfo
	for rowNumber := 1; ; rowNumber++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || !recordHasRequiredFields(header, record) {
			continue
		}
		session, ok := parseUsageRecord(baseName, rowNumber, header, record)
		if !ok {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func headerIndex(record []string) map[string]int {
	index := make(map[string]int, len(record))
	for i, value := range record {
		trimmed := strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
		if trimmed == "" {
			continue
		}
		index[trimmed] = i
	}
	return index
}

func parseUsageRecord(baseName string, rowNumber int, header map[string]int, record []string) (provider.SessionInfo, bool) {
	value := func(name string) string {
		idx, ok := header[name]
		if !ok || idx >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[idx])
	}

	ts, err := time.Parse(time.RFC3339Nano, value(cursorDateHeader))
	if err != nil {
		return provider.SessionInfo{}, false
	}

	inputCacheCreate, err := parseCursorInt(value(cursorInputCacheCreateHeader))
	if err != nil {
		return provider.SessionInfo{}, false
	}
	inputOther, err := parseCursorInt(value(cursorInputOtherHeader))
	if err != nil {
		return provider.SessionInfo{}, false
	}
	inputCacheRead, err := parseCursorInt(value(cursorInputCacheReadHeader))
	if err != nil {
		return provider.SessionInfo{}, false
	}
	output, err := parseCursorInt(value(cursorOutputHeader))
	if err != nil {
		return provider.SessionInfo{}, false
	}

	kind := value(cursorKindHeader)
	model := value(cursorModelHeader)
	title := strings.TrimSpace(strings.TrimSpace(kind) + " " + strings.TrimSpace(model))
	if title == "" {
		title = "Cursor usage export"
	}

	return provider.SessionInfo{
		ProviderName: "cursor",
		ModelName:    model,
		SessionID:    baseName + ":" + strconv.Itoa(rowNumber),
		Title:        title,
		StartTime:    ts,
		EndTime:      ts,
		Turns:        1,
		TokenUsage: provider.TokenUsage{
			InputOther:       inputOther,
			Output:           output,
			InputCacheRead:   inputCacheRead,
			InputCacheCreate: inputCacheCreate,
		},
	}, true
}

func recordHasRequiredFields(header map[string]int, record []string) bool {
	for _, name := range cursorRequiredHeaders {
		if header[name] >= len(record) {
			return false
		}
	}
	return true
}

func parseCursorInt(value string) (int, error) {
	value = strings.ReplaceAll(value, ",", "")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
