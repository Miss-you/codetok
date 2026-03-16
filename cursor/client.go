package cursor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://cursor.com"

// ValidationResult reports the result of validating a Cursor session token.
type ValidationResult struct {
	Valid          bool
	MembershipType string
	Message        string
}

// Client performs explicit Cursor dashboard API requests.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a Cursor API client with an optional base URL override.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: httpClient,
	}
}

type usageSummaryResponse struct {
	BillingCycleStart string `json:"billingCycleStart"`
	BillingCycleEnd   string `json:"billingCycleEnd"`
	MembershipType    string `json:"membershipType"`
}

// ValidateSession checks whether the supplied Cursor session token is accepted.
func (c *Client) ValidateSession(ctx context.Context, token string) (ValidationResult, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/api/usage-summary", token, nil)
	if err != nil {
		return ValidationResult{}, err
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return ValidationResult{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ValidationResult{Message: "Cursor session token expired or invalid"}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ValidationResult{}, fmt.Errorf("cursor API returned status %s", resp.Status)
	}

	var body usageSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ValidationResult{}, fmt.Errorf("decode usage summary: %w", err)
	}
	if body.BillingCycleStart == "" || body.BillingCycleEnd == "" {
		return ValidationResult{Message: "invalid response format from Cursor usage summary"}, nil
	}

	return ValidationResult{
		Valid:          true,
		MembershipType: body.MembershipType,
	}, nil
}

// FetchUsageCSV downloads the Cursor usage CSV export for the active account.
func (c *Client) FetchUsageCSV(ctx context.Context, token string) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/api/dashboard/export-usage-events-csv?strategy=tokens", token, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("cursor session token expired or invalid")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cursor API returned status %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read usage CSV: %w", err)
	}
	trimmed := bytes.TrimSpace(data)
	// Cursor may prepend a UTF-8 BOM to CSV downloads.
	trimmed = bytes.TrimPrefix(trimmed, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(trimmed, []byte("Date,")) {
		return nil, fmt.Errorf("invalid response from Cursor API: expected CSV data")
	}
	return data, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) newRequest(ctx context.Context, method, path, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cookie", "WorkosCursorSessionToken="+token)
	req.Header.Set("Referer", "https://www.cursor.com/settings")
	req.Header.Set("User-Agent", "codetok")
	return req, nil
}
