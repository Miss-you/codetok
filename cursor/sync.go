package cursor

import (
	"context"
	"errors"
)

// SyncResult describes a completed local CSV sync.
type SyncResult struct {
	Path  string
	Bytes int
}

// Sync fetches the Cursor usage CSV and atomically updates the local cache.
func (s *Service) Sync(ctx context.Context) (SyncResult, error) {
	creds, err := s.Store.LoadCredentials()
	if err != nil {
		if errors.Is(err, ErrNoCredentials) {
			return SyncResult{}, ErrNoCredentials
		}
		return SyncResult{}, err
	}

	data, err := s.Client.FetchUsageCSV(ctx, creds.SessionToken)
	if err != nil {
		return SyncResult{}, err
	}

	path, err := s.Store.WriteSyncedCSV(data)
	if err != nil {
		return SyncResult{}, err
	}

	return SyncResult{
		Path:  path,
		Bytes: len(data),
	}, nil
}
