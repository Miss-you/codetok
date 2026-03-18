package cursor

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/miss-you/codetok/internal/testutil"
)

type scoredCommitFixture = testutil.CursorActivityRow

func TestReadActivity_AggregatesComposerAndTabMetrics(t *testing.T) {
	dbPath := writeActivityFixtureDB(t, []scoredCommitFixture{
		{ComposerAdded: 12, ComposerDeleted: 3, TabAdded: 4, TabDeleted: 1},
		{ComposerAdded: 5, ComposerDeleted: 2, TabAdded: 6, TabDeleted: 0},
	})

	reader := NewActivityReader()
	result, err := reader.Read(dbPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if !result.HasData {
		t.Fatal("expected result to report available activity data")
	}
	if result.DBPath != dbPath {
		t.Fatalf("DBPath = %q, want %q", result.DBPath, dbPath)
	}
	if result.ScoredCommits != 2 {
		t.Fatalf("ScoredCommits = %d, want 2", result.ScoredCommits)
	}
	if result.Composer.LinesAdded != 17 {
		t.Fatalf("Composer.LinesAdded = %d, want 17", result.Composer.LinesAdded)
	}
	if result.Composer.LinesDeleted != 5 {
		t.Fatalf("Composer.LinesDeleted = %d, want 5", result.Composer.LinesDeleted)
	}
	if result.Tab.LinesAdded != 10 {
		t.Fatalf("Tab.LinesAdded = %d, want 10", result.Tab.LinesAdded)
	}
	if result.Tab.LinesDeleted != 1 {
		t.Fatalf("Tab.LinesDeleted = %d, want 1", result.Tab.LinesDeleted)
	}
}

func TestReadActivity_MissingDatabaseReturnsNoData(t *testing.T) {
	reader := NewActivityReader()
	dbPath := filepath.Join(t.TempDir(), "missing.db")

	result, err := reader.Read(dbPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if result.HasData {
		t.Fatal("expected missing database to report no data")
	}
	if result.DBPath != dbPath {
		t.Fatalf("DBPath = %q, want %q", result.DBPath, dbPath)
	}
	if result.ScoredCommits != 0 {
		t.Fatalf("ScoredCommits = %d, want 0", result.ScoredCommits)
	}
	if result.Composer.LinesAdded != 0 || result.Composer.LinesDeleted != 0 {
		t.Fatalf("composer metrics = %+v, want zeros", result.Composer)
	}
	if result.Tab.LinesAdded != 0 || result.Tab.LinesDeleted != 0 {
		t.Fatalf("tab metrics = %+v, want zeros", result.Tab)
	}
}

func TestReadActivity_ComposerOnlyKeepsTabZero(t *testing.T) {
	dbPath := writeActivityFixtureDB(t, []scoredCommitFixture{
		{ComposerAdded: 8, ComposerDeleted: 2},
	})

	reader := NewActivityReader()
	result, err := reader.Read(dbPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if !result.HasData {
		t.Fatal("expected composer-only database to report data")
	}
	if result.Composer.LinesAdded != 8 || result.Composer.LinesDeleted != 2 {
		t.Fatalf("composer metrics = %+v, want added=8 deleted=2", result.Composer)
	}
	if result.Tab.LinesAdded != 0 || result.Tab.LinesDeleted != 0 {
		t.Fatalf("tab metrics = %+v, want zeros", result.Tab)
	}
}

func TestReadActivity_TabOnlyKeepsComposerZero(t *testing.T) {
	dbPath := writeActivityFixtureDB(t, []scoredCommitFixture{
		{TabAdded: 11, TabDeleted: 4},
	})

	reader := NewActivityReader()
	result, err := reader.Read(dbPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if !result.HasData {
		t.Fatal("expected tab-only database to report data")
	}
	if result.Tab.LinesAdded != 11 || result.Tab.LinesDeleted != 4 {
		t.Fatalf("tab metrics = %+v, want added=11 deleted=4", result.Tab)
	}
	if result.Composer.LinesAdded != 0 || result.Composer.LinesDeleted != 0 {
		t.Fatalf("composer metrics = %+v, want zeros", result.Composer)
	}
}

func TestReadActivity_UnexpectedStatErrorReturnsError(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(parentFile, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	reader := NewActivityReader()
	_, err := reader.Read(filepath.Join(parentFile, "ai-code-tracking.db"))
	if err == nil {
		t.Fatal("expected unexpected stat failure to return error")
	}
}

func TestReadActivity_OpenDatabaseErrorReturnsError(t *testing.T) {
	dbPath := writeActivityFixtureDB(t, []scoredCommitFixture{
		{ComposerAdded: 1},
	})

	wantErr := errors.New("open db failed")
	reader := &ActivityReader{
		openDB: func(driverName, dataSourceName string) (*sql.DB, error) {
			if driverName != activityDriverName {
				t.Fatalf("driverName = %q, want %q", driverName, activityDriverName)
			}
			if dataSourceName != dbPath {
				t.Fatalf("dataSourceName = %q, want %q", dataSourceName, dbPath)
			}
			return nil, wantErr
		},
	}

	_, err := reader.Read(dbPath)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Read error = %v, want %v", err, wantErr)
	}
}

func TestReadActivity_UnexpectedQueryErrorReturnsError(t *testing.T) {
	dbPath := writeActivityFixtureDB(t, []scoredCommitFixture{
		{ComposerAdded: 1},
	})

	reader := &ActivityReader{
		openDB: func(driverName, dataSourceName string) (*sql.DB, error) {
			db, err := sql.Open(driverName, dataSourceName)
			if err != nil {
				t.Fatalf("sql.Open returned error: %v", err)
			}
			if err := db.Close(); err != nil {
				t.Fatalf("db.Close returned error: %v", err)
			}
			return db, nil
		},
	}

	_, err := reader.Read(dbPath)
	if err == nil {
		t.Fatal("expected unexpected query failure to return error")
	}
	if got := err.Error(); got == "" || got == "no such table: scored_commits" {
		t.Fatalf("Read error = %q, want wrapped unexpected query error", got)
	}
}

func writeActivityFixtureDB(t *testing.T, rows []scoredCommitFixture) string {
	return testutil.WriteCursorActivityDB(t, rows)
}
