package cursor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreSaveCredentialsWritesRestrictedFile(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.SaveCredentials(Credentials{SessionToken: "token-123"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	path, err := store.CredentialsPath()
	if err != nil {
		t.Fatalf("CredentialsPath returned error: %v", err)
	}
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

func TestStoreCredentialsPathReturnsDefaultRootError(t *testing.T) {
	oldUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) {
		return "", errors.New("home lookup failed")
	}
	t.Cleanup(func() {
		userHomeDir = oldUserHomeDir
	})

	_, err := NewStore("").CredentialsPath()
	if err == nil {
		t.Fatal("expected CredentialsPath to return a root resolution error")
	}
	if !strings.Contains(err.Error(), "home lookup failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStoreSyncedCSVPathReturnsDefaultRootError(t *testing.T) {
	oldUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) {
		return "", errors.New("home lookup failed")
	}
	t.Cleanup(func() {
		userHomeDir = oldUserHomeDir
	})

	_, err := NewStore("").SyncedCSVPath()
	if err == nil {
		t.Fatal("expected SyncedCSVPath to return a root resolution error")
	}
	if !strings.Contains(err.Error(), "home lookup failed") {
		t.Fatalf("unexpected error: %v", err)
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

func TestStoreWriteSyncedCSVOverwritesExistingFileWhenRenameNeedsReplace(t *testing.T) {
	store := NewStore(t.TempDir())

	path, err := store.WriteSyncedCSV([]byte("Date,Model\nold\n"))
	if err != nil {
		t.Fatalf("WriteSyncedCSV returned error: %v", err)
	}

	oldRenameFile := renameFile
	oldRemoveFile := removeFile
	oldRenameNeedsDestinationRemoval := renameNeedsDestinationRemoval
	renameCalls := 0
	renameFile = func(oldPath, newPath string) error {
		renameCalls++
		if renameCalls == 1 {
			return errors.New("destination already exists")
		}
		return os.Rename(oldPath, newPath)
	}
	removeFile = os.Remove
	renameNeedsDestinationRemoval = true
	t.Cleanup(func() {
		renameFile = oldRenameFile
		removeFile = oldRemoveFile
		renameNeedsDestinationRemoval = oldRenameNeedsDestinationRemoval
	})

	if _, err := store.WriteSyncedCSV([]byte("Date,Model\nnew\n")); err != nil {
		t.Fatalf("WriteSyncedCSV returned error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(got) != "Date,Model\nnew\n" {
		t.Fatalf("csv = %q, want overwritten content", string(got))
	}
	if renameCalls < 2 {
		t.Fatalf("renameCalls = %d, want retry after removing existing destination", renameCalls)
	}
}
