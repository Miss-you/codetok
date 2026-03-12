package cursor

import (
	"encoding/csv"
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

// Name returns the provider name.
func (p *Provider) Name() string {
	return "cursor"
}

// CollectSessions scans baseDir for Cursor usage export CSV files and returns one
// session-like record per CSV row.
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	if baseDir == "" {
		var err error
		baseDir, err = defaultCursorDir()
		if err != nil {
			return nil, err
		}
	}

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

func defaultCursorDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codetok", "cursor"), nil
}

func parseUsageCSV(path string) ([]provider.SessionInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	header := headerIndex(records[0])
	required := []string{
		cursorDateHeader,
		cursorKindHeader,
		cursorModelHeader,
		cursorInputCacheCreateHeader,
		cursorInputOtherHeader,
		cursorInputCacheReadHeader,
		cursorOutputHeader,
	}
	for _, name := range required {
		if _, ok := header[name]; !ok {
			return nil, os.ErrInvalid
		}
	}

	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	sessions := make([]provider.SessionInfo, 0, len(records)-1)
	for i, record := range records[1:] {
		rowNumber := i + 1
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

func parseCursorInt(value string) (int, error) {
	value = strings.ReplaceAll(value, ",", "")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
