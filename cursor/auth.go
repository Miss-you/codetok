package cursor

import (
	"context"
	"errors"
	"strings"
)

// APIClient abstracts the explicit remote calls used by Cursor auth and sync.
type APIClient interface {
	ValidateSession(ctx context.Context, token string) (ValidationResult, error)
	FetchUsageCSV(ctx context.Context, token string) ([]byte, error)
}

// StatusResult reports whether local credentials exist and whether they validate remotely.
type StatusResult struct {
	HasCredentials bool
	RemoteValid    bool
	MembershipType string
	Message        string
}

// Service coordinates local credential storage with explicit Cursor API access.
type Service struct {
	Store  Store
	Client APIClient
}

// NewService returns a Cursor service using the provided store and client.
func NewService(store Store, client APIClient) *Service {
	if client == nil {
		client = NewClient("", nil)
	}
	return &Service{
		Store:  store,
		Client: client,
	}
}

// Login validates the supplied token before saving it as the active credential.
func (s *Service) Login(ctx context.Context, token string) (ValidationResult, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return ValidationResult{}, errors.New("provide a Cursor session token")
	}

	result, err := s.Client.ValidateSession(ctx, token)
	if err != nil {
		return ValidationResult{}, err
	}
	if !result.Valid {
		if result.Message == "" {
			result.Message = "Cursor session token expired or invalid"
		}
		return result, errors.New(result.Message)
	}

	if err := s.Store.SaveCredentials(Credentials{SessionToken: token}); err != nil {
		return ValidationResult{}, err
	}
	return result, nil
}

// Status reports whether codetok has a local credential and whether it validates remotely.
func (s *Service) Status(ctx context.Context) (StatusResult, error) {
	creds, err := s.Store.LoadCredentials()
	if err != nil {
		if errors.Is(err, ErrNoCredentials) {
			return StatusResult{}, nil
		}
		return StatusResult{}, err
	}

	result, err := s.Client.ValidateSession(ctx, creds.SessionToken)
	if err != nil {
		return StatusResult{HasCredentials: true}, err
	}

	return StatusResult{
		HasCredentials: true,
		RemoteValid:    result.Valid,
		MembershipType: result.MembershipType,
		Message:        result.Message,
	}, nil
}

// Logout removes the active local credential.
func (s *Service) Logout() error {
	return s.Store.DeleteCredentials()
}
