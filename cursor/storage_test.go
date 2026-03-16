package cursor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveCredentialsWritesRestrictedFile(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.SaveCredentials(Credentials{SessionToken: "token-123"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	path := store.CredentialsPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("credential permissions = %#o, want 0600", got)
	}
}

func TestStoreSaveCredentialsFailureKeepsPreviousFile(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveCredentials(Credentials{SessionToken: "old-token"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	store.atomicWrite = func(string, []byte, os.FileMode) error {
		return errors.New("boom")
	}

	err := store.SaveCredentials(Credentials{SessionToken: "new-token"})
	if err == nil {
		t.Fatal("expected SaveCredentials to fail")
	}

	creds, loadErr := NewStore(store.RootDir).LoadCredentials()
	if loadErr != nil {
		t.Fatalf("LoadCredentials returned error: %v", loadErr)
	}
	if creds.SessionToken != "old-token" {
		t.Fatalf("SessionToken = %q, want old-token", creds.SessionToken)
	}
}

func TestStoreWriteSyncedCSVUsesSyncedSubdirectory(t *testing.T) {
	store := NewStore(t.TempDir())

	path, err := store.WriteSyncedCSV([]byte("Date,Model\n"))
	if err != nil {
		t.Fatalf("WriteSyncedCSV returned error: %v", err)
	}

	want := filepath.Join(store.RootDir, "synced", "usage.csv")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
}
