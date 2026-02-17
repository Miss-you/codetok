package provider

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseParallel_BasicConcurrency(t *testing.T) {
	items := make([]string, 20)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}

	parseFn := func(path string) (SessionInfo, error) {
		return SessionInfo{
			SessionID: path,
			Turns:     1,
		}, nil
	}

	results := ParseParallel(items, 4, parseFn)
	if len(results) != 20 {
		t.Errorf("expected 20 results, got %d", len(results))
	}

	// Verify all items were parsed (order may vary)
	seen := make(map[string]bool)
	for _, r := range results {
		seen[r.SessionID] = true
	}
	for _, item := range items {
		if !seen[item] {
			t.Errorf("missing result for %s", item)
		}
	}
}

func TestParseParallel_ErrorSkipped(t *testing.T) {
	items := []string{"ok-1", "fail-1", "ok-2", "fail-2", "ok-3"}

	parseFn := func(path string) (SessionInfo, error) {
		if path[0:4] == "fail" {
			return SessionInfo{}, fmt.Errorf("parse error for %s", path)
		}
		return SessionInfo{SessionID: path}, nil
	}

	results := ParseParallel(items, 2, parseFn)
	if len(results) != 3 {
		t.Errorf("expected 3 results (errors skipped), got %d", len(results))
	}

	seen := make(map[string]bool)
	for _, r := range results {
		seen[r.SessionID] = true
	}
	for _, expected := range []string{"ok-1", "ok-2", "ok-3"} {
		if !seen[expected] {
			t.Errorf("missing result for %s", expected)
		}
	}
}

func TestParseParallel_EmptyInput(t *testing.T) {
	results := ParseParallel(nil, 4, func(path string) (SessionInfo, error) {
		t.Error("parseFn should not be called for empty input")
		return SessionInfo{}, nil
	})
	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}

	results = ParseParallel([]string{}, 4, func(path string) (SessionInfo, error) {
		t.Error("parseFn should not be called for empty input")
		return SessionInfo{}, nil
	})
	if results != nil {
		t.Errorf("expected nil for empty slice, got %v", results)
	}
}

func TestParseParallel_WorkerLimit(t *testing.T) {
	const maxWorkers = 3
	const totalItems = 20

	var concurrent atomic.Int64
	var maxConcurrent atomic.Int64

	items := make([]string, totalItems)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}

	parseFn := func(path string) (SessionInfo, error) {
		cur := concurrent.Add(1)
		// Track the maximum observed concurrency
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		// Sleep briefly to allow goroutines to overlap
		time.Sleep(5 * time.Millisecond)
		concurrent.Add(-1)
		return SessionInfo{SessionID: path}, nil
	}

	results := ParseParallel(items, maxWorkers, parseFn)
	if len(results) != totalItems {
		t.Errorf("expected %d results, got %d", totalItems, len(results))
	}

	observed := maxConcurrent.Load()
	if observed > int64(maxWorkers) {
		t.Errorf("concurrency exceeded limit: observed %d, limit %d", observed, maxWorkers)
	}
	if observed == 0 {
		t.Error("no concurrency observed; expected at least 1 concurrent worker")
	}
}

func TestParseParallel_SingleItem(t *testing.T) {
	results := ParseParallel([]string{"only-one"}, 4, func(path string) (SessionInfo, error) {
		return SessionInfo{SessionID: path}, nil
	})
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].SessionID != "only-one" {
		t.Errorf("expected SessionID 'only-one', got %q", results[0].SessionID)
	}
}

func TestParseParallel_DefaultWorkers(t *testing.T) {
	// With maxWorkers=0 it should use the default and still work correctly
	items := []string{"a", "b", "c"}
	results := ParseParallel(items, 0, func(path string) (SessionInfo, error) {
		return SessionInfo{SessionID: path}, nil
	})
	if len(results) != 3 {
		t.Errorf("expected 3 results with default workers, got %d", len(results))
	}
}
