package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// CursorActivityRow represents one scored_commits fixture row for Cursor activity tests.
type CursorActivityRow struct {
	ComposerAdded   int
	ComposerDeleted int
	TabAdded        int
	TabDeleted      int
}

// WriteCursorActivityDB creates a temporary Cursor activity SQLite database for tests.
func WriteCursorActivityDB(t testing.TB, rows []CursorActivityRow) string {
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
			cursorActivityCommitHash(i),
			"main",
			1000+i,
			row.ComposerAdded+row.TabAdded,
			row.ComposerDeleted+row.TabDeleted,
			row.TabAdded,
			row.TabDeleted,
			row.ComposerAdded,
			row.ComposerDeleted,
		)
		if err != nil {
			t.Fatalf("inserting fixture row %d: %v", i, err)
		}
	}

	return dbPath
}

func cursorActivityCommitHash(i int) string {
	return "commit-" + string(rune('a'+i))
}
