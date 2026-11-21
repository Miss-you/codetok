package testutil

import (
	"database/sql"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// AntigravityCall is one model call in an Antigravity conversation fixture.
type AntigravityCall struct {
	Model      string
	Time       time.Time
	Input      uint64
	Output     uint64
	CacheRead  uint64
	CacheWrite uint64
}

// WriteAntigravityRoot creates an Antigravity data root with one conversation
// database per entry of convs (conversation ID -> calls) and returns the root.
func WriteAntigravityRoot(t testing.TB, convs map[string][]AntigravityCall) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "conversations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for id, calls := range convs {
		db, err := sql.Open("sqlite", filepath.Join(dir, id+".db"))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []string{
			"CREATE TABLE steps (idx integer PRIMARY KEY, metadata blob)",
			"CREATE TABLE gen_metadata (idx integer PRIMARY KEY, data blob)",
		} {
			if _, err := db.Exec(s); err != nil {
				t.Fatal(err)
			}
		}
		for i, c := range calls {
			step := uint64(i + 1)
			ts := cat(pbUint(1, uint64(c.Time.Unix())), pbUint(2, uint64(c.Time.Nanosecond())))
			if _, err := db.Exec("INSERT INTO steps VALUES (?, ?)", step, pbBytes(1, ts)); err != nil {
				t.Fatal(err)
			}
			usage := cat(pbUint(2, c.Input), pbUint(3, c.Output), pbUint(4, c.CacheWrite), pbUint(5, c.CacheRead))
			chat := cat(pbBytes(4, usage), pbBytes(19, []byte(c.Model)))
			gen := cat(pbBytes(1, chat), pbBytes(2, binary.AppendUvarint(nil, step)))
			if _, err := db.Exec("INSERT INTO gen_metadata VALUES (?, ?)", i, gen); err != nil {
				t.Fatal(err)
			}
		}
		db.Close()
	}
	return root
}

func pbUint(num int, v uint64) []byte {
	return binary.AppendUvarint(binary.AppendUvarint(nil, uint64(num<<3)), v)
}

func pbBytes(num int, b []byte) []byte {
	out := binary.AppendUvarint(binary.AppendUvarint(nil, uint64(num<<3|2)), uint64(len(b)))
	return append(out, b...)
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
