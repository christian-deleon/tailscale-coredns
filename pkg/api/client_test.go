package api

import (
	"os"
	"reflect"
	"testing"
)

func TestGetTailnetFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		tailnet  string
		expected string
	}{
		{
			name:     "valid organization - domain format",
			tailnet:  "mydomain.com",
			expected: "mydomain.com",
		},
		{
			name:     "valid organization - email format",
			tailnet:  "name@mydomain.com",
			expected: "name@mydomain.com",
		},
		{
			name:     "valid organization - with spaces",
			tailnet:  "  mydomain.com  ",
			expected: "mydomain.com",
		},
		{
			name:     "empty tailnet uses default",
			tailnet:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("TS_TAILNET")

			if tt.tailnet != "" {
				os.Setenv("TS_TAILNET", tt.tailnet)
			}
			defer os.Unsetenv("TS_TAILNET")

			result, err := GetTailnetFromEnv()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name             string
		clientID         string
		clientSecret     string
		tailnet          string
		expectedTailnet  string
	}{
		{
			name:            "with explicit tailnet",
			clientID:        "test-client-id",
			clientSecret:    "test-client-secret",
			tailnet:         "test-tailnet",
			expectedTailnet: "test-tailnet",
		},
		{
			name:            "empty tailnet uses default",
			clientID:        "test-client-id",
			clientSecret:    "test-client-secret",
			tailnet:         "",
			expectedTailnet: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.clientID, tt.clientSecret, tt.tailnet)

			if client.clientID != tt.clientID {
				t.Errorf("Expected clientID to be '%s', got %s", tt.clientID, client.clientID)
			}

			if client.clientSecret != tt.clientSecret {
				t.Errorf("Expected clientSecret to be '%s', got %s", tt.clientSecret, client.clientSecret)
			}

			if client.tailnet != tt.expectedTailnet {
				t.Errorf("Expected tailnet to be '%s', got %s", tt.expectedTailnet, client.tailnet)
			}

			if client.httpClient == nil {
				t.Error("Expected httpClient to be initialized")
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		wantErr  bool
	}{
		{
			name:    "valid domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			domain:  "sub.example.com",
			wantErr: false,
		},
		{
			name:    "valid multi-level subdomain",
			domain:  "a.b.c.example.com",
			wantErr: false,
		},
		{
			name:    "domain with spaces",
			domain:  "  example.com  ",
			wantErr: false,
		},
		{
			name:    "empty domain",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "single word",
			domain:  "localhost",
			wantErr: true,
		},
		{
			name:    "domain with empty part",
			domain:  "example..com",
			wantErr: true,
		},
		{
			name:    "domain ending with dot",
			domain:  "example.com.",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single domain",
			input:    "example.com",
			expected: []string{"example.com"},
			wantErr:  false,
		},
		{
			name:     "multiple domains",
			input:    "example.com,test.org,demo.net",
			expected: []string{"example.com", "test.org", "demo.net"},
			wantErr:  false,
		},
		{
			name:     "domains with spaces",
			input:    "example.com, test.org , demo.net",
			expected: []string{"example.com", "test.org", "demo.net"},
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid domain in list",
			input:    "example.com,invalid,test.org",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "empty domains in list",
			input:    "example.com,,test.org",
			expected: []string{"example.com", "test.org"},
			wantErr:  false,
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDomains(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDomains() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseDomains() = %v, want %v", result, tt.expected)
			}
		})
	}
}