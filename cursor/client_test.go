package cursor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientValidateSession_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/usage-summary" {
			t.Fatalf("path = %q, want /api/usage-summary", r.URL.Path)
		}
		if cookie := r.Header.Get("Cookie"); !strings.Contains(cookie, "WorkosCursorSessionToken=token-123") {
			t.Fatalf("Cookie header = %q, want session token", cookie)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"billingCycleStart":"2026-03-01T00:00:00Z","billingCycleEnd":"2026-04-01T00:00:00Z","membershipType":"pro"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.ValidateSession(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("ValidateSession returned error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected token to validate")
	}
	if result.MembershipType != "pro" {
		t.Fatalf("MembershipType = %q, want pro", result.MembershipType)
	}
}

func TestClientValidateSession_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.ValidateSession(context.Background(), "bad-token")
	if err != nil {
		t.Fatalf("ValidateSession returned error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid token result")
	}
	if !strings.Contains(strings.ToLower(result.Message), "invalid") {
		t.Fatalf("Message = %q, want invalid-token explanation", result.Message)
	}
}

func TestClientFetchUsageCSV_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dashboard/export-usage-events-csv" {
			t.Fatalf("path = %q, want export endpoint", r.URL.Path)
		}
		if got := r.URL.Query().Get("strategy"); got != "tokens" {
			t.Fatalf("strategy = %q, want tokens", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("Date,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	csv, err := client.FetchUsageCSV(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("FetchUsageCSV returned error: %v", err)
	}
	if !strings.HasPrefix(string(csv), "Date,") {
		t.Fatalf("csv = %q, want Date header", string(csv))
	}
}

func TestClientFetchUsageCSV_AcceptsUTF8BOM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("\ufeffDate,Model,Input (w/ Cache Write),Input (w/o Cache Write),Cache Read,Output Tokens\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	csv, err := client.FetchUsageCSV(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("FetchUsageCSV returned error: %v", err)
	}
	if !strings.Contains(string(csv), "Date,Model") {
		t.Fatalf("csv = %q, want CSV header with Date column", string(csv))
	}
}

func TestClientFetchUsageCSV_RejectsInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"not csv"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.FetchUsageCSV(context.Background(), "token-123")
	if err == nil {
		t.Fatal("expected invalid response error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "csv") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientFetchUsageCSV_PropagatesNetworkError(t *testing.T) {
	httpClient := &http.Client{
		Timeout: 50 * time.Millisecond,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial error")
		}),
	}

	client := NewClient("https://cursor.example.test", httpClient)
	_, err := client.FetchUsageCSV(context.Background(), "token-123")
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "dial error") {
		t.Fatalf("unexpected error: %v", err)
	}
}
