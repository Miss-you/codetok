package cursor

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

var ErrNoCredentials = errors.New("cursor credentials not found")

// Credentials stores the active Cursor session token saved by codetok.
type Credentials struct {
	SessionToken string    `json:"session_token"`
	SavedAt      time.Time `json:"saved_at"`
}

// Store manages Cursor credentials and synced CSV cache files.
type Store struct {
	RootDir     string
	atomicWrite func(path string, data []byte, perm os.FileMode) error
}

// NewStore returns a Store rooted at rootDir. An empty rootDir resolves to the
// default ~/.codetok/cursor path when methods are called.
func NewStore(rootDir string) Store {
	return Store{
		RootDir:     rootDir,
		atomicWrite: atomicWriteFile,
	}
}

// DefaultRootDir returns the default codetok-owned Cursor storage root.
func DefaultRootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codetok", "cursor"), nil
}

// CredentialsPath returns the path to the saved credential file.
func (s Store) CredentialsPath() string {
	path, _ := s.credentialsPath()
	return path
}

// SyncedCSVPath returns the path to the tool-owned synced CSV cache file.
func (s Store) SyncedCSVPath() string {
	path, _ := s.syncedCSVPath()
	return path
}

func (s Store) credentialsPath() (string, error) {
	root, err := s.rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "credentials.json"), nil
}

func (s Store) syncedCSVPath() (string, error) {
	root, err := s.rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "synced", "usage.csv"), nil
}

func (s Store) rootDir() (string, error) {
	if s.RootDir != "" {
		return s.RootDir, nil
	}
	return DefaultRootDir()
}

func (s Store) writer() func(string, []byte, os.FileMode) error {
	if s.atomicWrite != nil {
		return s.atomicWrite
	}
	return atomicWriteFile
}

// SaveCredentials persists the active Cursor session token with owner-only file access.
func (s Store) SaveCredentials(creds Credentials) error {
	if creds.SessionToken == "" {
		return errors.New("cursor session token is required")
	}
	if creds.SavedAt.IsZero() {
		creds.SavedAt = time.Now().UTC()
	}

	path, err := s.credentialsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return s.writer()(path, data, 0o600)
}

// LoadCredentials loads the saved Cursor session token.
func (s Store) LoadCredentials() (Credentials, error) {
	path, err := s.credentialsPath()
	if err != nil {
		return Credentials{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Credentials{}, ErrNoCredentials
		}
		return Credentials{}, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return Credentials{}, err
	}
	if creds.SessionToken == "" {
		return Credentials{}, ErrNoCredentials
	}
	return creds, nil
}

// DeleteCredentials removes the saved Cursor session token, if present.
func (s Store) DeleteCredentials() error {
	path, err := s.credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WriteSyncedCSV atomically updates the tool-owned synced CSV cache file.
func (s Store) WriteSyncedCSV(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("cursor sync returned an empty CSV payload")
	}

	path, err := s.syncedCSVPath()
	if err != nil {
		return "", err
	}
	if err := s.writer()(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) (err error) {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(parent, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}
