package provider

import (
	"strings"
	"testing"
)

func TestEachJSONLLine_ReadsMultiMBLines(t *testing.T) {
	long := strings.Repeat("x", 3*1024*1024)
	input := "first\n" + long + "\nlast\n"
	var lines []string
	if err := EachJSONLLine(strings.NewReader(input), func(line []byte) {
		lines = append(lines, string(line))
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 3 || lines[0] != "first" || len(lines[1]) != len(long) || lines[2] != "last" {
		t.Fatalf("unexpected lines: count=%d", len(lines))
	}
}

func TestEachJSONLLine_RejectsPathologicalLine(t *testing.T) {
	input := strings.Repeat("x", maxJSONLLineSize+1) + "\n"
	called := false
	err := EachJSONLLine(strings.NewReader(input), func(line []byte) {
		called = true
	})
	if err == nil {
		t.Fatal("expected an error for a line beyond the size limit")
	}
	if called {
		t.Error("fn must not be called for an over-limit line")
	}
}

func TestEachJSONLLine_HandlesFinalLineWithoutNewline(t *testing.T) {
	var lines []string
	if err := EachJSONLLine(strings.NewReader("a\nb"), func(line []byte) {
		lines = append(lines, string(line))
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}
