package provider

import (
	"os"
	"runtime"
	"strconv"
	"sync"
)

// ParseFunc is a function that parses a single item and returns a SessionInfo.
type ParseFunc func(path string) (SessionInfo, error)

// ParseParallel parses multiple items in parallel with bounded concurrency.
// maxWorkers <= 0 means use default (from CODETOK_WORKERS env or runtime.NumCPU()).
// Items that return errors are silently skipped.
func ParseParallel(items []string, maxWorkers int, parseFn ParseFunc) []SessionInfo {
	if maxWorkers <= 0 {
		maxWorkers = defaultWorkers()
	}
	if maxWorkers > len(items) {
		maxWorkers = len(items)
	}
	if len(items) == 0 {
		return nil
	}

	var mu sync.Mutex
	var results []SessionInfo
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()
			info, err := parseFn(path)
			if err != nil {
				return // skip failed items
			}
			mu.Lock()
			results = append(results, info)
			mu.Unlock()
		}(item)
	}
	wg.Wait()
	return results
}

func defaultWorkers() int {
	if s := os.Getenv("CODETOK_WORKERS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	n := runtime.NumCPU()
	if n > 8 {
		return 8 // cap at 8 to avoid too much file I/O contention
	}
	return n
}
