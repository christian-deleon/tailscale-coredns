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

// SplitDNSConfig represents the split DNS configuration as a map from domains to nameservers
type SplitDNSConfig map[string][]string

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

	// Use "-" for default tailnet if not specified
	if tailnet == "" {
		tailnet = "-"
	}

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
func (a *Client) GetSplitDNS(ctx context.Context) (SplitDNSConfig, error) {
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
		return make(SplitDNSConfig), nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get split DNS failed with status: %d", resp.StatusCode)
	}

	var splitDNS SplitDNSConfig
	if err := json.NewDecoder(resp.Body).Decode(&splitDNS); err != nil {
		return nil, fmt.Errorf("failed to decode split DNS response: %w", err)
	}

	return splitDNS, nil
}

// PatchSplitDNS performs a partial update of split DNS configuration
func (a *Client) PatchSplitDNS(ctx context.Context, updates SplitDNSConfig) error {
	token, err := a.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/dns/split-dns", a.tailnet)

	body, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal split DNS updates: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to patch split DNS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("patch split DNS failed with status: %d", resp.StatusCode)
	}

	return nil
}

// PutSplitDNS replaces the entire split DNS configuration
func (a *Client) PutSplitDNS(ctx context.Context, config SplitDNSConfig) error {
	token, err := a.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/dns/split-dns", a.tailnet)

	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal split DNS config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to put split DNS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("put split DNS failed with status: %d", resp.StatusCode)
	}

	return nil
}

// AddIPToDomains adds an IP to the specified domains in split DNS
func (a *Client) AddIPToDomains(ctx context.Context, domains []string, ip string) error {
	// Get current configuration
	currentConfig, err := a.GetSplitDNS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current split DNS: %w", err)
	}

	// Prepare updates
	updates := make(SplitDNSConfig)

	for _, domain := range domains {
		nameservers := currentConfig[domain]

		// Check if IP already exists
		found := false
		for _, ns := range nameservers {
			if ns == ip {
				found = true
				break
			}
		}

		// Add IP if not found
		if !found {
			nameservers = append(nameservers, ip)
			updates[domain] = nameservers
		}
	}

	// Apply updates if any
	if len(updates) > 0 {
		return a.PatchSplitDNS(ctx, updates)
	}

	return nil
}

// RemoveIPFromDomains removes an IP from the specified domains in split DNS
func (a *Client) RemoveIPFromDomains(ctx context.Context, domains []string, ip string) error {
	// Get current configuration
	currentConfig, err := a.GetSplitDNS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current split DNS: %w", err)
	}

	// Prepare updates
	updates := make(SplitDNSConfig)

	for _, domain := range domains {
		nameservers := currentConfig[domain]

		// Filter out the IP
		var newNameservers []string
		for _, ns := range nameservers {
			if ns != ip {
				newNameservers = append(newNameservers, ns)
			}
		}

		// Only update if IP was actually removed
		if len(newNameservers) != len(nameservers) {
			if len(newNameservers) == 0 {
				// Use null to clear the domain if no nameservers left
				updates[domain] = nil
			} else {
				updates[domain] = newNameservers
			}
		}
	}

	// Apply updates if any
	if len(updates) > 0 {
		return a.PatchSplitDNS(ctx, updates)
	}

	return nil
}

// GetTailnetFromEnv gets the tailnet from environment variables
func GetTailnetFromEnv() (string, error) {
	// Check if tailnet is explicitly set first
	if tailnet := os.Getenv("TS_TAILNET"); tailnet != "" {
		return strings.TrimSpace(tailnet), nil
	}

	// Return empty string to use default "-"
	return "", nil
}

// ValidateDomain validates that a domain has at least a second-level domain and TLD
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Remove trailing dot if present
	domain = strings.TrimSuffix(domain, ".")

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return fmt.Errorf("domain must have at least a second-level domain and TLD (e.g., example.com)")
	}

	// Check each part is not empty
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("domain has empty part at position %d", i)
		}
	}

	return nil
}

// ParseDomains parses a comma-separated list of domains and validates each one
func ParseDomains(domainsStr string) ([]string, error) {
	if domainsStr == "" {
		return nil, fmt.Errorf("domains string cannot be empty")
	}

	parts := strings.Split(domainsStr, ",")
	domains := make([]string, 0, len(parts))

	for _, part := range parts {
		domain := strings.TrimSpace(part)
		if domain == "" {
			continue
		}

		if err := ValidateDomain(domain); err != nil {
			return nil, fmt.Errorf("invalid domain '%s': %w", domain, err)
		}

		domains = append(domains, domain)
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("no valid domains found")
	}

	return domains, nil
}