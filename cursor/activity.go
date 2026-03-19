package cursor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const activityDriverName = "sqlite"

const activityTotalsQuery = `
SELECT
	COUNT(*),
	COALESCE(SUM(composerLinesAdded), 0),
	COALESCE(SUM(composerLinesDeleted), 0),
	COALESCE(SUM(tabLinesAdded), 0),
	COALESCE(SUM(tabLinesDeleted), 0)
FROM scored_commits
`

type openActivityDBFunc func(driverName, dataSourceName string) (*sql.DB, error)

// ActivityMetric holds line-based Cursor attribution data for one activity source.
type ActivityMetric struct {
	LinesAdded   int `json:"lines_added"`
	LinesDeleted int `json:"lines_deleted"`
}

// ActivityResult keeps Cursor line attribution separate from token accounting.
type ActivityResult struct {
	DBPath        string         `json:"db_path"`
	HasData       bool           `json:"has_data"`
	ScoredCommits int            `json:"scored_commits"`
	Composer      ActivityMetric `json:"composer"`
	Tab           ActivityMetric `json:"tab"`
}

// ActivityReader reads Cursor activity attribution from the local tracking database.
type ActivityReader struct {
	openDB openActivityDBFunc
}

// NewActivityReader returns a reader for Cursor's local activity attribution database.
func NewActivityReader() *ActivityReader {
	return &ActivityReader{openDB: sql.Open}
}

// DefaultActivityDBPath returns the default Cursor tracking database path.
func DefaultActivityDBPath() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor", "ai-tracking", "ai-code-tracking.db"), nil
}

// Read aggregates composer and tab activity without reusing token fields.
func (r *ActivityReader) Read(dbPath string) (ActivityResult, error) {
	resolvedPath, err := resolveActivityDBPath(dbPath)
	if err != nil {
		return ActivityResult{}, err
	}

	result := ActivityResult{DBPath: resolvedPath}
	if _, err := os.Stat(resolvedPath); err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return result, nil
		}
		return result, fmt.Errorf("stat cursor activity database %q: %w", resolvedPath, err)
	}

	openDB := r.openDB
	if openDB == nil {
		openDB = sql.Open
	}

	db, err := openDB(activityDriverName, resolvedPath)
	if err != nil {
		return result, fmt.Errorf("open cursor activity database %q: %w", resolvedPath, err)
	}
	defer db.Close()

	err = db.QueryRow(activityTotalsQuery).Scan(
		&result.ScoredCommits,
		&result.Composer.LinesAdded,
		&result.Composer.LinesDeleted,
		&result.Tab.LinesAdded,
		&result.Tab.LinesDeleted,
	)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return result, nil
		}
		return result, fmt.Errorf("query cursor activity database %q: %w", resolvedPath, err)
	}

	result.HasData = result.ScoredCommits > 0
	return result, nil
}

func resolveActivityDBPath(dbPath string) (string, error) {
	if strings.TrimSpace(dbPath) != "" {
		return dbPath, nil
	}
	return DefaultActivityDBPath()
}

// Activity returns local Cursor line-attribution data without entering token paths.
func (s *Service) Activity(_ context.Context, dbPath string) (ActivityResult, error) {
	return NewActivityReader().Read(dbPath)
}
