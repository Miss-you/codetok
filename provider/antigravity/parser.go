// Package antigravity reads token usage from Antigravity CLI (agy) and IDE
// conversation databases. See openspec/changes/antigravity-usage-support/design.md
// for the protobuf field mapping.
package antigravity

import (
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/miss-you/codetok/provider"
)

const providerName = "antigravity"

func init() {
	provider.Register(&Provider{})
}

// Provider implements provider.Provider for Antigravity.
type Provider struct{}

// Name returns the provider name.
func (p *Provider) Name() string {
	return providerName
}

// Field numbers, read from the descriptors embedded in the agy binary.
const (
	genChatModel   = 1 // CortexStepGeneratorMetadata.chat_model
	genStepIndices = 2 // CortexStepGeneratorMetadata.step_indices

	chatUsage         = 4  // ChatModelMetadata.usage
	chatResponseModel = 19 // ChatModelMetadata.response_model
	chatDisplayModel  = 21 // ChatModelMetadata.model_display_name

	usageInput      = 2 // ModelUsageStats.input_tokens (excludes cache reads)
	usageOutput     = 3 // ModelUsageStats.output_tokens (includes thinking)
	usageCacheWrite = 4 // ModelUsageStats.cache_write_tokens
	usageCacheRead  = 5 // ModelUsageStats.cache_read_tokens

	stepCreatedAt          = 1 // CortexStepMetadata.created_at
	stepFinishedGenerating = 7 // CortexStepMetadata.finished_generating_at

	timestampSeconds = 1
	timestampNanos   = 2
)

// conversation is one conversations/<cascade_id>.db file.
type conversation struct {
	path  string
	title string
}

// CollectSessions returns one session per conversation database with counted usage.
func (p *Provider) CollectSessions(baseDir string) ([]provider.SessionInfo, error) {
	convs, err := discover(baseDir)
	if err != nil {
		return nil, err
	}
	byPath := indexByPath(convs)
	return provider.ParseParallel(paths(convs), 0, func(path string) (provider.SessionInfo, error) {
		events, err := parseConversation(byPath[path])
		if err != nil {
			return provider.SessionInfo{}, err
		}
		if len(events) == 0 {
			return provider.SessionInfo{}, fmt.Errorf("no usage in %s", path)
		}
		return sessionFromEvents(events), nil
	}), nil
}

// CollectUsageEvents returns one usage event per counted model call.
func (p *Provider) CollectUsageEvents(baseDir string) ([]provider.UsageEvent, error) {
	return p.CollectUsageEventsInRange(baseDir, provider.UsageEventCollectOptions{})
}

// CollectUsageEventsInRange skips conversation files last written before opts.Since.
func (p *Provider) CollectUsageEventsInRange(baseDir string, opts provider.UsageEventCollectOptions) ([]provider.UsageEvent, error) {
	convs, err := discover(baseDir)
	if err != nil {
		return nil, err
	}

	var selected []string
	for _, c := range convs {
		if opts.Metrics != nil {
			opts.Metrics.ConsideredFiles++
		}
		if opts.HasRange() && opts.ShouldSkipFileByModTime(lastWrite(c.path)) {
			if opts.Metrics != nil {
				opts.Metrics.SkippedFiles++
			}
			continue
		}
		selected = append(selected, c.path)
	}
	if opts.Metrics != nil {
		opts.Metrics.ParsedFiles += len(selected)
	}

	byPath := indexByPath(convs)
	events := provider.ParseUsageEventsParallel(selected, 0, func(path string) ([]provider.UsageEvent, error) {
		return parseConversation(byPath[path])
	})
	if opts.Metrics != nil {
		opts.Metrics.EmittedEvents += len(events)
	}
	return events, nil
}

// defaultRoots are the CLI and IDE data roots.
func defaultRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".gemini", "antigravity-cli"),
		filepath.Join(home, ".gemini", "antigravity"),
	}
}

// discover lists conversation databases. An explicit baseDir must exist;
// missing default roots are skipped.
func discover(baseDir string) ([]conversation, error) {
	roots := defaultRoots()
	if baseDir != "" {
		if _, err := os.Stat(baseDir); err != nil {
			return nil, err
		}
		roots = []string{baseDir}
	}

	var convs []conversation
	for _, root := range roots {
		matches, err := filepath.Glob(filepath.Join(root, "conversations", "*.db"))
		if err != nil {
			return nil, fmt.Errorf("listing antigravity conversations in %s: %w", root, err)
		}
		if len(matches) == 0 {
			continue
		}
		titles := loadTitles(filepath.Join(root, "conversation_summaries.db"))
		for _, m := range matches {
			convs = append(convs, conversation{path: m, title: titles[sessionID(m)]})
		}
	}
	return convs, nil
}

func indexByPath(convs []conversation) map[string]conversation {
	m := make(map[string]conversation, len(convs))
	for _, c := range convs {
		m[c.path] = c
	}
	return m
}

func paths(convs []conversation) []string {
	out := make([]string, len(convs))
	for i, c := range convs {
		out[i] = c.path
	}
	return out
}

func sessionID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".db")
}

// lastWrite is the later mtime of the database and its WAL.
func lastWrite(path string) time.Time {
	var t time.Time
	for _, p := range []string{path, path + "-wal"} {
		if info, err := os.Stat(p); err == nil && info.ModTime().After(t) {
			t = info.ModTime()
		}
	}
	return t
}

// loadTitles maps conversation IDs to titles. Titles are optional, so errors
// yield an empty map.
func loadTitles(path string) map[string]string {
	titles := map[string]string{}
	if _, err := os.Stat(path); err != nil {
		return titles
	}
	db, cleanup, err := openReadOnly(path)
	if err != nil {
		return titles
	}
	defer cleanup()
	rows, err := db.Query("SELECT conversation_id, title FROM conversation_summaries")
	if err != nil {
		return titles
	}
	defer rows.Close()
	for rows.Next() {
		var id, title sql.NullString
		if rows.Scan(&id, &title) == nil && id.Valid {
			titles[id.String] = title.String
		}
	}
	return titles
}

// openReadOnly opens an Antigravity SQLite file without writing to or locking
// it. With an empty WAL the file is opened immutable; otherwise the database
// and WAL are copied to a temp directory so the latest writes are included.
func openReadOnly(path string) (*sql.DB, func(), error) {
	wal, err := os.Stat(path + "-wal")
	if err != nil || wal.Size() == 0 {
		db, err := sql.Open("sqlite", sqliteURI(path, "immutable=1"))
		if err != nil {
			return nil, nil, fmt.Errorf("opening %s: %w", path, err)
		}
		return db, func() { db.Close() }, nil
	}

	dir, err := os.MkdirTemp("", "codetok-antigravity-")
	if err != nil {
		return nil, nil, fmt.Errorf("creating temp dir for %s: %w", path, err)
	}
	removeDir := func() { os.RemoveAll(dir) }
	copyPath := filepath.Join(dir, filepath.Base(path))
	for _, suffix := range []string{"", "-wal"} {
		if err := copyFile(path+suffix, copyPath+suffix); err != nil {
			removeDir()
			return nil, nil, fmt.Errorf("copying %s: %w", path+suffix, err)
		}
	}
	db, err := sql.Open("sqlite", sqliteURI(copyPath, ""))
	if err != nil {
		removeDir()
		return nil, nil, fmt.Errorf("opening copy of %s: %w", path, err)
	}
	return db, func() { db.Close(); removeDir() }, nil
}

func sqliteURI(path, query string) string {
	return (&url.URL{Scheme: "file", Path: path, RawQuery: query}).String()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// parseConversation returns one event per gen_metadata row with non-zero usage
// and a timestamp. Undecodable rows are skipped.
func parseConversation(c conversation) ([]provider.UsageEvent, error) {
	db, cleanup, err := openReadOnly(c.path)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	stepTimes, err := loadStepTimes(db)
	if err != nil {
		return nil, fmt.Errorf("reading steps from %s: %w", c.path, err)
	}

	rows, err := db.Query("SELECT idx, data FROM gen_metadata ORDER BY idx")
	if err != nil {
		return nil, fmt.Errorf("reading gen_metadata from %s: %w", c.path, err)
	}
	defer rows.Close()

	id := sessionID(c.path)
	var events []provider.UsageEvent
	for rows.Next() {
		var idx int64
		var data []byte
		if err := rows.Scan(&idx, &data); err != nil {
			continue
		}
		usage, model, steps, ok := decodeGeneration(data)
		if !ok || len(steps) == 0 {
			continue
		}
		ts, ok := stepTimes[int64(steps[0])]
		if !ok {
			continue
		}
		events = append(events, provider.UsageEvent{
			ProviderName: providerName,
			ModelName:    model,
			SessionID:    id,
			Title:        c.title,
			Timestamp:    ts,
			TokenUsage:   usage,
			SourcePath:   c.path,
			EventID:      fmt.Sprintf("%s:%d", id, idx),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading gen_metadata from %s: %w", c.path, err)
	}
	return events, nil
}

// loadStepTimes maps step indices to created_at, falling back to finished_generating_at.
func loadStepTimes(db *sql.DB) (map[int64]time.Time, error) {
	rows, err := db.Query("SELECT idx, metadata FROM steps")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	times := map[int64]time.Time{}
	for rows.Next() {
		var idx int64
		var md []byte
		if err := rows.Scan(&idx, &md); err != nil || len(md) == 0 {
			continue
		}
		fields, err := parseProto(md)
		if err != nil {
			continue
		}
		for _, num := range []int{stepCreatedAt, stepFinishedGenerating} {
			if ts, ok := decodeTimestamp(fields, num); ok {
				times[idx] = ts
				break
			}
		}
	}
	return times, rows.Err()
}

func decodeTimestamp(fields []protoField, num int) (time.Time, bool) {
	b, ok := message(fields, num)
	if !ok {
		return time.Time{}, false
	}
	ts, err := parseProto(b)
	if err != nil {
		return time.Time{}, false
	}
	sec := uintField(ts, timestampSeconds)
	if sec == 0 {
		return time.Time{}, false
	}
	return time.Unix(int64(sec), int64(uintField(ts, timestampNanos))), true
}

// decodeGeneration reads usage, model and step indices from a
// CortexStepGeneratorMetadata blob. ok is false when the row has no usable usage.
func decodeGeneration(data []byte) (usage provider.TokenUsage, model string, steps []uint64, ok bool) {
	top, err := parseProto(data)
	if err != nil {
		return usage, "", nil, false
	}
	chatBytes, found := message(top, genChatModel)
	if !found {
		return usage, "", nil, false
	}
	chat, err := parseProto(chatBytes)
	if err != nil {
		return usage, "", nil, false
	}
	usageBytes, found := message(chat, chatUsage)
	if !found {
		return usage, "", nil, false
	}
	u, err := parseProto(usageBytes)
	if err != nil {
		return usage, "", nil, false
	}
	usage = provider.TokenUsage{
		InputOther:       int(uintField(u, usageInput)),
		Output:           int(uintField(u, usageOutput)),
		InputCacheRead:   int(uintField(u, usageCacheRead)),
		InputCacheCreate: int(uintField(u, usageCacheWrite)),
	}
	if usage.Total() == 0 {
		return usage, "", nil, false
	}
	steps, err = repeatedUint(top, genStepIndices)
	if err != nil {
		return usage, "", nil, false
	}
	for _, num := range []int{chatResponseModel, chatDisplayModel} {
		if b, found := message(chat, num); found && len(b) > 0 {
			model = string(b)
			break
		}
	}
	return usage, model, steps, true
}

// sessionFromEvents sums one conversation's events. The session model is the
// most frequent call model (ties broken by name).
func sessionFromEvents(events []provider.UsageEvent) provider.SessionInfo {
	s := provider.SessionInfo{
		ProviderName: providerName,
		SessionID:    events[0].SessionID,
		Title:        events[0].Title,
		Turns:        len(events),
	}
	counts := map[string]int{}
	for _, e := range events {
		s.TokenUsage.InputOther += e.TokenUsage.InputOther
		s.TokenUsage.Output += e.TokenUsage.Output
		s.TokenUsage.InputCacheRead += e.TokenUsage.InputCacheRead
		s.TokenUsage.InputCacheCreate += e.TokenUsage.InputCacheCreate
		if s.StartTime.IsZero() || e.Timestamp.Before(s.StartTime) {
			s.StartTime = e.Timestamp
		}
		if e.Timestamp.After(s.EndTime) {
			s.EndTime = e.Timestamp
		}
		if e.ModelName != "" {
			counts[e.ModelName]++
		}
	}
	models := make([]string, 0, len(counts))
	for m := range counts {
		models = append(models, m)
	}
	sort.Slice(models, func(i, j int) bool {
		if counts[models[i]] != counts[models[j]] {
			return counts[models[i]] > counts[models[j]]
		}
		return models[i] < models[j]
	})
	if len(models) > 0 {
		s.ModelName = models[0]
	}
	return s
}
