package cursor

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestServiceSyncWritesCSVToLocalCache(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveCredentials(Credentials{SessionToken: "token-123"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	svc := NewService(store, &stubAPIClient{
		fetchCSV: []byte("Date,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens\n"),
	})

	result, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Path != store.SyncedCSVPath() {
		t.Fatalf("Path = %q, want %q", result.Path, store.SyncedCSVPath())
	}

	got, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(got) != "Date,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens\n" {
		t.Fatalf("csv = %q, want synced csv", string(got))
	}
}

func TestServiceSyncPreservesExistingCacheOnFailure(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveCredentials(Credentials{SessionToken: "token-123"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}
	if _, err := store.WriteSyncedCSV([]byte("Date,Model\nold\n")); err != nil {
		t.Fatalf("WriteSyncedCSV returned error: %v", err)
	}

	svc := NewService(store, &stubAPIClient{
		fetchErr: errors.New("network down"),
	})

	if _, err := svc.Sync(context.Background()); err == nil {
		t.Fatal("expected Sync to fail")
	}

	got, err := os.ReadFile(store.SyncedCSVPath())
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(got) != "Date,Model\nold\n" {
		t.Fatalf("csv = %q, want previous cache to remain", string(got))
	}
}
