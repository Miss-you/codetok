package cursor

import (
	"context"
	"errors"
	"testing"
)

type stubAPIClient struct {
	validateResult ValidationResult
	validateErr    error
	fetchCSV       []byte
	fetchErr       error
}

func (s *stubAPIClient) ValidateSession(_ context.Context, token string) (ValidationResult, error) {
	if token == "" {
		return ValidationResult{}, errors.New("missing token")
	}
	return s.validateResult, s.validateErr
}

func (s *stubAPIClient) FetchUsageCSV(context.Context, string) ([]byte, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return append([]byte(nil), s.fetchCSV...), nil
}

func TestServiceLoginSavesValidatedCredentials(t *testing.T) {
	store := NewStore(t.TempDir())
	svc := NewService(store, &stubAPIClient{
		validateResult: ValidationResult{Valid: true, MembershipType: "pro"},
	})

	result, err := svc.Login(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected validated login result")
	}

	creds, err := store.LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials returned error: %v", err)
	}
	if creds.SessionToken != "token-123" {
		t.Fatalf("SessionToken = %q, want token-123", creds.SessionToken)
	}
}

func TestServiceLoginRejectsInvalidCredentials(t *testing.T) {
	store := NewStore(t.TempDir())
	svc := NewService(store, &stubAPIClient{
		validateResult: ValidationResult{Valid: false, Message: "Session token expired or invalid"},
	})

	_, err := svc.Login(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected login to reject invalid credentials")
	}

	if _, err := store.LoadCredentials(); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("LoadCredentials error = %v, want ErrNoCredentials", err)
	}
}

func TestServiceStatusReturnsLoggedOutWithoutCredentials(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), &stubAPIClient{})

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.HasCredentials {
		t.Fatal("expected no saved credentials")
	}
	if status.RemoteValid {
		t.Fatal("expected remote validation to be false when logged out")
	}
}

func TestServiceStatusSeparatesSavedCredentialsFromRemoteValidity(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveCredentials(Credentials{SessionToken: "saved-token"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	svc := NewService(store, &stubAPIClient{
		validateResult: ValidationResult{Valid: false, Message: "Session token expired or invalid"},
	})

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.HasCredentials {
		t.Fatal("expected saved credentials to be reported")
	}
	if status.RemoteValid {
		t.Fatal("expected remote validity to be false")
	}
}

func TestServiceLogoutRemovesSavedCredentials(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.SaveCredentials(Credentials{SessionToken: "saved-token"}); err != nil {
		t.Fatalf("SaveCredentials returned error: %v", err)
	}

	svc := NewService(store, &stubAPIClient{})
	if err := svc.Logout(); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	if _, err := store.LoadCredentials(); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("LoadCredentials error = %v, want ErrNoCredentials", err)
	}
}
