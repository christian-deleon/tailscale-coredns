package api

import (
	"os"
	"testing"
)

func TestGetTailnetFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
		hasError bool
	}{
		{
			name:     "valid domain",
			domain:   "mydomain.com",
			expected: "mydomain",
			hasError: false,
		},
		{
			name:     "valid domain with subdomain",
			domain:   "sub.mydomain.com",
			expected: "sub",
			hasError: false,
		},
		{
			name:     "invalid domain - no dot",
			domain:   "mydomain",
			expected: "",
			hasError: true,
		},
		{
			name:     "empty domain",
			domain:   "",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.domain != "" {
				os.Setenv("TS_DOMAIN", tt.domain)
			} else {
				os.Unsetenv("TS_DOMAIN")
			}

			result, err := GetTailnetFromEnv()

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s but got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("test-client-id", "test-client-secret", "test-tailnet")

	if client.clientID != "test-client-id" {
		t.Errorf("Expected clientID to be 'test-client-id', got %s", client.clientID)
	}

	if client.clientSecret != "test-client-secret" {
		t.Errorf("Expected clientSecret to be 'test-client-secret', got %s", client.clientSecret)
	}

	if client.tailnet != "test-tailnet" {
		t.Errorf("Expected tailnet to be 'test-tailnet', got %s", client.tailnet)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}