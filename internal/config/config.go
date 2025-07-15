package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the tailscale-coredns service
type Config struct {
	// Tailscale OAuth credentials
	ClientID     string
	ClientSecret string

	// DNS configuration
	Domain      string
	Hostname    string
	HostsFile   string
	ForwardTo   string

	// Split DNS settings
	EnableSplitDNS bool
	Tailnet        string

	// Tailscale settings
	Ephemeral bool

	// Refresh interval
	RefreshInterval int

	// Additional configuration
	AdditionalConfig string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{}

	// Required: OAuth credentials
	config.ClientID = os.Getenv("TS_CLIENT_ID")
	config.ClientSecret = os.Getenv("TS_CLIENT_SECRET")

	if config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("TS_CLIENT_ID and TS_CLIENT_SECRET are required")
	}

	// Required: Domain and hostname
	config.Domain = os.Getenv("TS_DOMAIN")
	if config.Domain == "" {
		return nil, fmt.Errorf("TS_DOMAIN is required")
	}

	config.Hostname = os.Getenv("TS_HOSTNAME")
	if config.Hostname == "" {
		return nil, fmt.Errorf("TS_HOSTNAME is required")
	}

	// Optional: Hosts file
	config.HostsFile = os.Getenv("TS_HOSTS_FILE")
	if config.HostsFile == "" {
		// Check for default hosts file
		defaultHostsFile := "/etc/ts-dns/hosts/custom_hosts"
		if fileExists(defaultHostsFile) {
			config.HostsFile = defaultHostsFile
		}
	}

	// Optional: Forward server
	config.ForwardTo = os.Getenv("TS_FORWARD_TO")
	if config.ForwardTo == "" {
		config.ForwardTo = "/etc/resolv.conf"
	}

	// Optional: Split DNS
	config.EnableSplitDNS = strings.ToLower(os.Getenv("TS_ENABLE_SPLIT_DNS")) == "true"
	
	// Optional: Tailnet name (auto-detected if not set)
	if tailnet := os.Getenv("TS_TAILNET"); tailnet != "" {
		// Normalize the tailnet name to handle cases like "tailXYZ.ts.net" -> "tailXYZ"
		config.Tailnet = normalizeTailnetName(tailnet)
	}

	// Optional: Ephemeral mode
	ephemeralStr := os.Getenv("TS_EPHEMERAL")
	if ephemeralStr == "" {
		config.Ephemeral = true // Default to true
	} else {
		config.Ephemeral = strings.ToLower(ephemeralStr) == "true"
	}

	// Optional: Refresh interval
	if intervalStr := os.Getenv("TSC_REFRESH_INTERVAL"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
			config.RefreshInterval = interval
		} else {
			return nil, fmt.Errorf("invalid TSC_REFRESH_INTERVAL value: %s", intervalStr)
		}
	} else {
		config.RefreshInterval = 30 // Default 30 seconds
	}

	// Optional: Additional configuration
	config.AdditionalConfig = os.Getenv("TS_ADDITIONAL_CONFIG")

	return config, nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// GetTailnet extracts the tailnet name from the domain
func (c *Config) GetTailnet() (string, error) {
	parts := strings.Split(c.Domain, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid domain format: %s", c.Domain)
	}
	return parts[0], nil
}

// Validate ensures all required configuration is present and valid
func (c *Config) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("TS_CLIENT_ID is required")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("TS_CLIENT_SECRET is required")
	}

	if c.Domain == "" {
		return fmt.Errorf("TS_DOMAIN is required")
	}

	if c.Hostname == "" {
		return fmt.Errorf("TS_HOSTNAME is required")
	}

	if c.RefreshInterval <= 0 {
		return fmt.Errorf("refresh interval must be positive")
	}

	// Validate hosts file exists if specified
	if c.HostsFile != "" && !fileExists(c.HostsFile) {
		return fmt.Errorf("hosts file does not exist: %s", c.HostsFile)
	}

	return nil
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