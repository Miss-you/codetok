package cursor

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

type scoredCommitFixture struct {
	composerAdded   int
	composerDeleted int
	tabAdded        int
	tabDeleted      int
}

func TestReadActivity_AggregatesComposerAndTabMetrics(t *testing.T) {
	dbPath := writeActivityFixtureDB(t, []scoredCommitFixture{
		{composerAdded: 12, composerDeleted: 3, tabAdded: 4, tabDeleted: 1},
		{composerAdded: 5, composerDeleted: 2, tabAdded: 6, tabDeleted: 0},
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
		{composerAdded: 8, composerDeleted: 2},
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
		{tabAdded: 11, tabDeleted: 4},
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

func writeActivityFixtureDB(t *testing.T, rows []scoredCommitFixture) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ai-code-tracking.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE scored_commits (
	commitHash TEXT NOT NULL,
	branchName TEXT NOT NULL,
	scoredAt INTEGER NOT NULL,
	linesAdded INTEGER,
	linesDeleted INTEGER,
	tabLinesAdded INTEGER,
	tabLinesDeleted INTEGER,
	composerLinesAdded INTEGER,
	composerLinesDeleted INTEGER,
	humanLinesAdded INTEGER,
	humanLinesDeleted INTEGER,
	blankLinesAdded INTEGER,
	blankLinesDeleted INTEGER,
	commitMessage TEXT,
	commitDate TEXT,
	v1AiPercentage TEXT,
	v2AiPercentage TEXT,
	PRIMARY KEY (commitHash, branchName)
);
`)
	if err != nil {
		t.Fatalf("creating scored_commits table: %v", err)
	}

	for i, row := range rows {
		_, err := db.Exec(`
INSERT INTO scored_commits (
	commitHash, branchName, scoredAt, linesAdded, linesDeleted,
	tabLinesAdded, tabLinesDeleted, composerLinesAdded, composerLinesDeleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
			commitHashForIndex(i),
			"main",
			1000+i,
			row.composerAdded+row.tabAdded,
			row.composerDeleted+row.tabDeleted,
			row.tabAdded,
			row.tabDeleted,
			row.composerAdded,
			row.composerDeleted,
		)
		if err != nil {
			t.Fatalf("inserting fixture row %d: %v", i, err)
		}
	}

	return dbPath
}

func commitHashForIndex(i int) string {
	return "commit-" + string(rune('a'+i))
}
