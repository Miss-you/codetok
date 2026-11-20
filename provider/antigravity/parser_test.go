package antigravity

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/miss-you/codetok/provider"
)

// --- protobuf encoding helpers ---

func pbKey(num, wire int) []byte { return binary.AppendUvarint(nil, uint64(num<<3|wire)) }

func pbUint(num int, v uint64) []byte { return binary.AppendUvarint(pbKey(num, 0), v) }

func pbBytes(num int, b []byte) []byte {
	out := binary.AppendUvarint(pbKey(num, 2), uint64(len(b)))
	return append(out, b...)
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

type genSpec struct {
	input, output, cacheRead, cacheWrite uint64
	responseModel, displayModel          string
	steps                                []uint64
	unpackedSteps                        bool
}

func encodeGeneration(g genSpec) []byte {
	usage := cat(pbUint(1, 1525), pbUint(usageInput, g.input), pbUint(usageOutput, g.output),
		pbUint(usageCacheWrite, g.cacheWrite), pbUint(usageCacheRead, g.cacheRead),
		pbUint(9, g.output/2), pbUint(10, g.output-g.output/2))
	chat := pbBytes(chatUsage, usage)
	if g.responseModel != "" {
		chat = append(chat, pbBytes(chatResponseModel, []byte(g.responseModel))...)
	}
	if g.displayModel != "" {
		chat = append(chat, pbBytes(chatDisplayModel, []byte(g.displayModel))...)
	}
	top := pbBytes(genChatModel, chat)
	if g.unpackedSteps {
		for _, s := range g.steps {
			top = append(top, pbUint(genStepIndices, s)...)
		}
	} else if len(g.steps) > 0 {
		var packed []byte
		for _, s := range g.steps {
			packed = binary.AppendUvarint(packed, s)
		}
		top = append(top, pbBytes(genStepIndices, packed)...)
	}
	return append(top, pbBytes(4, []byte("exec-1"))...)
}

func encodeTimestamp(t time.Time) []byte {
	return cat(pbUint(timestampSeconds, uint64(t.Unix())), pbUint(timestampNanos, uint64(t.Nanosecond())))
}

// encodeStep builds CortexStepMetadata; the step also carries model_usage (field 9)
// like real data, which the provider must not count.
func encodeStep(created, finished time.Time) []byte {
	var out []byte
	if !created.IsZero() {
		out = append(out, pbBytes(stepCreatedAt, encodeTimestamp(created))...)
	}
	if !finished.IsZero() {
		out = append(out, pbBytes(stepFinishedGenerating, encodeTimestamp(finished))...)
	}
	return append(out, pbBytes(9, cat(pbUint(usageInput, 999), pbUint(usageOutput, 999)))...)
}

// --- fixture databases ---

type fixtureConv struct {
	id    string
	steps map[int64][]byte
	gens  [][]byte
}

func createConversationDB(t *testing.T, path string, conv fixtureConv) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	stmts := []string{
		"CREATE TABLE steps (idx integer PRIMARY KEY, step_type integer NOT NULL DEFAULT 0, metadata blob)",
		"CREATE TABLE gen_metadata (idx integer PRIMARY KEY, data blob, size integer NOT NULL DEFAULT 0)",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	for idx, md := range conv.steps {
		if _, err := db.Exec("INSERT INTO steps (idx, metadata) VALUES (?, ?)", idx, md); err != nil {
			t.Fatal(err)
		}
	}
	for i, g := range conv.gens {
		if _, err := db.Exec("INSERT INTO gen_metadata (idx, data) VALUES (?, ?)", i, g); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func writeRoot(t *testing.T, convs []fixtureConv, titles map[string]string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "conversations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range convs {
		createConversationDB(t, filepath.Join(dir, c.id+".db"), c).Close()
	}
	if titles != nil {
		db, err := sql.Open("sqlite", filepath.Join(root, "conversation_summaries.db"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("CREATE TABLE conversation_summaries (conversation_id text PRIMARY KEY, title text NOT NULL DEFAULT '')"); err != nil {
			t.Fatal(err)
		}
		for id, title := range titles {
			if _, err := db.Exec("INSERT INTO conversation_summaries VALUES (?, ?)", id, title); err != nil {
				t.Fatal(err)
			}
		}
		db.Close()
	}
	return root
}

var (
	t1 = time.Date(2026, 11, 19, 23, 50, 0, 0, time.UTC)
	t2 = time.Date(2026, 11, 20, 0, 10, 0, 500, time.UTC)
)

func sampleConv() fixtureConv {
	return fixtureConv{
		id: "conv-a",
		steps: map[int64][]byte{
			1: encodeStep(t1, time.Time{}),
			2: encodeStep(t1, time.Time{}),
			3: encodeStep(time.Time{}, t2), // created_at missing: falls back
			4: encodeStep(time.Time{}, time.Time{}),
		},
		gens: [][]byte{
			// one call that produced steps 1 and 2: counted once
			encodeGeneration(genSpec{input: 100, output: 40, cacheRead: 1000, responseModel: "gemini-4-hydrogen", steps: []uint64{1, 2}}),
			// unpacked step indices, display-model fallback, cache write
			encodeGeneration(genSpec{input: 10, output: 5, cacheRead: 2000, cacheWrite: 7, displayModel: "Gemini 4 Argon", steps: []uint64{3}, unpackedSteps: true}),
			// zero usage: skipped
			encodeGeneration(genSpec{steps: []uint64{3}}),
			// no step indices: skipped
			encodeGeneration(genSpec{input: 1, output: 1}),
			// step without any timestamp: skipped
			encodeGeneration(genSpec{input: 1, output: 1, steps: []uint64{4}}),
			// step that does not exist: skipped
			encodeGeneration(genSpec{input: 1, output: 1, steps: []uint64{99}}),
			// malformed blob: skipped
			{0xff, 0xff, 0xff},
		},
	}
}

func TestCollectUsageEvents(t *testing.T) {
	root := writeRoot(t, []fixtureConv{sampleConv()}, map[string]string{"conv-a": "Fix the parser"})

	events, err := (&Provider{}).CollectUsageEvents(root)
	if err != nil {
		t.Fatalf("CollectUsageEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(events), events)
	}
	byModel := map[string]provider.UsageEvent{}
	for _, e := range events {
		byModel[e.ModelName] = e
	}

	first := byModel["gemini-4-hydrogen"]
	if want := (provider.TokenUsage{InputOther: 100, Output: 40, InputCacheRead: 1000}); first.TokenUsage != want {
		t.Errorf("first usage = %+v, want %+v", first.TokenUsage, want)
	}
	if !first.Timestamp.Equal(t1) {
		t.Errorf("first timestamp = %v, want %v", first.Timestamp, t1)
	}
	if first.ProviderName != "antigravity" || first.SessionID != "conv-a" || first.Title != "Fix the parser" {
		t.Errorf("first metadata = %+v", first)
	}
	if first.EventID != "conv-a:0" {
		t.Errorf("first EventID = %q", first.EventID)
	}

	second := byModel["Gemini 4 Argon"]
	if want := (provider.TokenUsage{InputOther: 10, Output: 5, InputCacheRead: 2000, InputCacheCreate: 7}); second.TokenUsage != want {
		t.Errorf("second usage = %+v, want %+v", second.TokenUsage, want)
	}
	if !second.Timestamp.Equal(t2) {
		t.Errorf("second timestamp = %v, want finished_generating_at %v", second.Timestamp, t2)
	}
}

func TestCollectSessions(t *testing.T) {
	extra := fixtureConv{
		id:    "conv-b",
		steps: map[int64][]byte{1: encodeStep(t1, time.Time{})},
		gens: [][]byte{
			encodeGeneration(genSpec{input: 1, output: 1, responseModel: "gemini-4-neon", steps: []uint64{1}}),
			encodeGeneration(genSpec{input: 2, output: 2, responseModel: "gemini-4-carbon", steps: []uint64{1}}),
			encodeGeneration(genSpec{input: 3, output: 3, responseModel: "gemini-4-carbon", steps: []uint64{1}}),
		},
	}
	empty := fixtureConv{id: "conv-empty", steps: map[int64][]byte{}, gens: nil}
	root := writeRoot(t, []fixtureConv{sampleConv(), extra, empty}, nil)

	sessions, err := (&Provider{}).CollectSessions(root)
	if err != nil {
		t.Fatalf("CollectSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2 (empty conversation dropped): %+v", len(sessions), sessions)
	}
	byID := map[string]provider.SessionInfo{}
	for _, s := range sessions {
		byID[s.SessionID] = s
	}

	a := byID["conv-a"]
	if want := (provider.TokenUsage{InputOther: 110, Output: 45, InputCacheRead: 3000, InputCacheCreate: 7}); a.TokenUsage != want {
		t.Errorf("conv-a usage = %+v, want %+v", a.TokenUsage, want)
	}
	if a.Turns != 2 || !a.StartTime.Equal(t1) || !a.EndTime.Equal(t2) {
		t.Errorf("conv-a turns/times = %d %v %v", a.Turns, a.StartTime, a.EndTime)
	}
	if b := byID["conv-b"]; b.ModelName != "gemini-4-carbon" || b.Turns != 3 {
		t.Errorf("conv-b model/turns = %q %d, want gemini-4-carbon 3", b.ModelName, b.Turns)
	}
}

func TestExplicitDirMustExist(t *testing.T) {
	_, err := (&Provider{}).CollectUsageEvents(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want not-exist", err)
	}
}

func TestRootWithoutConversations(t *testing.T) {
	events, err := (&Provider{}).CollectUsageEvents(t.TempDir())
	if err != nil || len(events) != 0 {
		t.Fatalf("events, err = %v, %v; want none, nil", events, err)
	}
}

func TestDefaultRootsUseHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cli := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(filepath.Join(cli, "conversations"), 0o755); err != nil {
		t.Fatal(err)
	}
	createConversationDB(t, filepath.Join(cli, "conversations", "conv-a.db"), sampleConv()).Close()
	// The IDE root does not exist; it is skipped.

	events, err := (&Provider{}).CollectUsageEvents("")
	if err != nil {
		t.Fatalf("CollectUsageEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
}

func TestReadsUncheckpointedWAL(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "conversations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "live.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, s := range []string{"PRAGMA journal_mode=WAL", "PRAGMA wal_autocheckpoint=0"} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	conv := sampleConv()
	for _, s := range []string{
		"CREATE TABLE steps (idx integer PRIMARY KEY, metadata blob)",
		"CREATE TABLE gen_metadata (idx integer PRIMARY KEY, data blob)",
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO steps VALUES (1, ?)", conv.steps[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO gen_metadata VALUES (0, ?)", conv.gens[0]); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path + "-wal"); err != nil || info.Size() == 0 {
		t.Fatalf("expected a non-empty WAL, got %v %v", info, err)
	}
	before, _ := os.ReadFile(path)

	events, err := (&Provider{}).CollectUsageEvents(root)
	if err != nil {
		t.Fatalf("CollectUsageEvents: %v", err)
	}
	if len(events) != 1 || events[0].TokenUsage.InputOther != 100 {
		t.Fatalf("events = %+v, want the WAL-only row", events)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("original database changed")
	}
}

func TestRangeSkipsOldFiles(t *testing.T) {
	root := writeRoot(t, []fixtureConv{sampleConv()}, nil)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(root, "conversations", "conv-a.db"), old, old); err != nil {
		t.Fatal(err)
	}

	metrics := &provider.UsageEventCollectMetrics{}
	events, err := (&Provider{}).CollectUsageEventsInRange(root, provider.UsageEventCollectOptions{
		Since:   time.Now().Add(-time.Hour),
		Metrics: metrics,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || metrics.ConsideredFiles != 1 || metrics.SkippedFiles != 1 || metrics.ParsedFiles != 0 {
		t.Fatalf("events=%d metrics=%+v", len(events), metrics)
	}
}

func TestParseProtoMalformed(t *testing.T) {
	cases := map[string][]byte{
		"truncated varint":   {0x08, 0xff},
		"length past end":    {0x0a, 0x05, 0x01},
		"field number zero":  {0x00, 0x01},
		"unsupported wire":   {0x0b},
		"truncated fixed64":  {0x09, 0x01, 0x02},
		"truncated fixed32":  {0x0d, 0x01},
		"truncated key only": {0x80},
	}
	for name, b := range cases {
		if _, err := parseProto(b); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, _, _, ok := decodeGeneration([]byte{0x0a, 0x02, 0xff, 0xff}); ok {
		t.Error("decodeGeneration accepted a malformed chat_model")
	}
}
