package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client handles Tailscale API operations
type Client struct {
	clientID     string
	clientSecret string
	tailnet      string
	httpClient   *http.Client
}

// SplitDNS represents the split DNS configuration
type SplitDNS struct {
	Domains []string `json:"domains"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewClient creates a new Tailscale API client
func NewClient(clientID, clientSecret, tailnet string) *Client {
	// Set environment variables for Tailscale CLI OAuth authentication
	os.Setenv("TS_CLIENT_ID", clientID)
	os.Setenv("TS_CLIENT_SECRET", clientSecret)

	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		tailnet:      tailnet,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// UpdateTailnet updates the tailnet name for this client
func (a *Client) UpdateTailnet(tailnet string) {
	a.tailnet = tailnet
}

// getAccessToken retrieves a short-lived OAuth token
func (a *Client) getAccessToken(ctx context.Context) (string, error) {
	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tailscale.com/api/v2/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// GetSplitDNS retrieves the current split DNS configuration
func (a *Client) GetSplitDNS(ctx context.Context) (*SplitDNS, error) {
	token, err := a.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/dns/split-dns", a.tailnet)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get split DNS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No split DNS configured yet
		return &SplitDNS{Domains: []string{}}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get split DNS failed with status: %d", resp.StatusCode)
	}

	var splitDNS SplitDNS
	if err := json.NewDecoder(resp.Body).Decode(&splitDNS); err != nil {
		return nil, fmt.Errorf("failed to decode split DNS response: %w", err)
	}

	return &splitDNS, nil
}

// UpdateSplitDNS updates the split DNS configuration
func (a *Client) UpdateSplitDNS(ctx context.Context, splitDNS *SplitDNS) error {
	token, err := a.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/dns/split-dns", a.tailnet)

	body, err := json.Marshal(splitDNS)
	if err != nil {
		return fmt.Errorf("failed to marshal split DNS: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update split DNS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update split DNS failed with status: %d", resp.StatusCode)
	}

	return nil
}

// AddDomainToSplitDNS adds a domain to the split DNS configuration
func (a *Client) AddDomainToSplitDNS(ctx context.Context, domain string) error {
	splitDNS, err := a.GetSplitDNS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current split DNS: %w", err)
	}

	// Check if domain already exists
	for _, existingDomain := range splitDNS.Domains {
		if existingDomain == domain {
			return nil // Domain already exists
		}
	}

	// Add the new domain
	splitDNS.Domains = append(splitDNS.Domains, domain)

	return a.UpdateSplitDNS(ctx, splitDNS)
}

// RemoveDomainFromSplitDNS removes a domain from the split DNS configuration
func (a *Client) RemoveDomainFromSplitDNS(ctx context.Context, domain string) error {
	splitDNS, err := a.GetSplitDNS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current split DNS: %w", err)
	}

	// Find and remove the domain
	var newDomains []string
	for _, existingDomain := range splitDNS.Domains {
		if existingDomain != domain {
			newDomains = append(newDomains, existingDomain)
		}
	}

	splitDNS.Domains = newDomains

	return a.UpdateSplitDNS(ctx, splitDNS)
}

// IsDomainInSplitDNS checks if a domain is currently in the split DNS configuration
func (a *Client) IsDomainInSplitDNS(ctx context.Context, domain string) (bool, error) {
	splitDNS, err := a.GetSplitDNS(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current split DNS: %w", err)
	}

	for _, existingDomain := range splitDNS.Domains {
		if existingDomain == domain {
			return true, nil
		}
	}

	return false, nil
}

// GetTailnetFromEnv extracts the tailnet name from the TS_DOMAIN environment variable
// or uses TS_TAILNET if explicitly set
func GetTailnetFromEnv() (string, error) {
	// Check if tailnet is explicitly set first
	if tailnet := os.Getenv("TS_TAILNET"); tailnet != "" {
		return normalizeTailnetName(tailnet), nil
	}

	domain := os.Getenv("TS_DOMAIN")
	if domain == "" {
		return "", fmt.Errorf("TS_DOMAIN environment variable is required")
	}

	// Extract tailnet from domain (e.g., "mydomain.com" -> "mydomain")
	// Note: This may not always be correct. Set TS_TAILNET explicitly if needed.
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid domain format: %s", domain)
	}

	return parts[0], nil
}

// normalizeTailnetName ensures we have just the tailnet name without .ts.net suffix
func normalizeTailnetName(tailnet string) string {
	// Remove .ts.net suffix if present
	if strings.HasSuffix(tailnet, ".ts.net") {
		tailnet = strings.TrimSuffix(tailnet, ".ts.net")
	}
	
	// If it's in format "hostname.tailnet.ts.net", extract just the tailnet part
	parts := strings.Split(tailnet, ".")
	if len(parts) >= 2 {
		// Return the last part which should be the tailnet name
		return parts[len(parts)-1]
	}
	
	return tailnet
}