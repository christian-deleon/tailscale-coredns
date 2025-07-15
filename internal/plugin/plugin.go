package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/netip"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"

	"tailscale-coredns/pkg/api"
)

type record struct {
	IPv4 net.IP
	IPv6 net.IP
}

type Tailscale struct {
	Next    plugin.Handler
	Domain  string
	records map[string]record
	mu      sync.RWMutex
	lc      *tailscale.LocalClient
	api     *api.Client
	// Split DNS management
	enableSplitDNS   bool
	splitDNSDomain   string
	ownIP            string
	lastVerifiedIP   string
	lastSplitDNSCheck time.Time
}

func New(domain string) (*Tailscale, error) {
	ts := &Tailscale{
		Domain:  domain,
		records: make(map[string]record),
		lc:      &tailscale.LocalClient{Socket: "/run/tailscale/tailscaled.sock"},
	}

	// Initialize split DNS if enabled
	if err := ts.initializeSplitDNS(); err != nil {
		clog.Errorf("Failed to initialize split DNS: %v", err)
		// Continue without split DNS if initialization fails
	}

	go ts.periodicRefresh()
	return ts, nil
}

// initializeSplitDNS sets up split DNS functionality if enabled
func (t *Tailscale) initializeSplitDNS() error {
	// Check if split DNS is enabled
	if os.Getenv("TS_ENABLE_SPLIT_DNS") != "true" {
		return nil
	}

	// Get OAuth credentials
	clientID := os.Getenv("TS_CLIENT_ID")
	clientSecret := os.Getenv("TS_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("TS_CLIENT_ID and TS_CLIENT_SECRET are required for split DNS")
	}

	// Try to get tailnet name from environment first, will be updated later from Tailscale status
	tailnet, err := api.GetTailnetFromEnv()
	if err != nil {
		clog.Warningf("Could not get tailnet from environment: %v", err)
		// Use domain prefix as fallback, will be corrected later
		parts := strings.Split(t.Domain, ".")
		if len(parts) > 0 {
			tailnet = parts[0]
		} else {
			tailnet = "unknown"
		}
	}

	// Create API client (tailnet will be updated when we have Tailscale status)
	t.api = api.NewClient(clientID, clientSecret, tailnet)
	t.enableSplitDNS = true
	t.splitDNSDomain = t.Domain

	clog.Infof("Split DNS enabled for domain: %s (tailnet will be auto-detected)", t.splitDNSDomain)
	return nil
}

// GetOwnIP retrieves the current node's Tailscale IP
func (t *Tailscale) GetOwnIP() (string, error) {
	ctx := context.Background()
	status, err := t.lc.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Tailscale status: %w", err)
	}

	if status == nil || status.Self == nil {
		return "", fmt.Errorf("no self node found in Tailscale status")
	}

	// Get the first IPv4 address
	for _, ip := range status.Self.TailscaleIPs {
		if ip.Is4() {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no IPv4 address found for self node")
}

// GetTailnetName retrieves the current tailnet name from Tailscale status
func (t *Tailscale) GetTailnetName() (string, error) {
	ctx := context.Background()
	status, err := t.lc.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Tailscale status: %w", err)
	}

	if status == nil || status.Self == nil {
		return "", fmt.Errorf("no self node found in Tailscale status")
	}

	// Extract tailnet from the DNSName (e.g., "host.tail326daa.ts.net." -> "tail326daa")
	dnsName := status.Self.DNSName
	if dnsName == "" {
		return "", fmt.Errorf("no DNS name found in Tailscale status")
	}

	// Remove trailing dot if present
	dnsName = strings.TrimSuffix(dnsName, ".")
	
	// Split by dots and extract the tailnet part
	parts := strings.Split(dnsName, ".")
	if len(parts) >= 3 && strings.HasSuffix(dnsName, ".ts.net") {
		// Format: hostname.tailnet.ts.net
		return parts[len(parts)-3], nil
	}

	return "", fmt.Errorf("could not extract tailnet from DNS name: %s", dnsName)
}

// AddToSplitDNS adds the current node's IP to split DNS
func (t *Tailscale) AddToSplitDNS() error {
	if !t.enableSplitDNS {
		return nil
	}

	// Get own IP
	ownIP, err := t.GetOwnIP()
	if err != nil {
		return fmt.Errorf("failed to get own IP: %w", err)
	}

	// Get the correct tailnet name from Tailscale status
	tailnet, err := t.GetTailnetName()
	if err != nil {
		clog.Warningf("Could not auto-detect tailnet: %v, using configured value", err)
	} else {
		// Update the API client with the correct tailnet
		t.api.UpdateTailnet(tailnet)
		clog.Debugf("Updated tailnet to: %s", tailnet)
	}

	t.ownIP = ownIP
	domain := t.splitDNSDomain

	clog.Infof("Adding domain %s with IP %s to split DNS", domain, ownIP)

	ctx := context.Background()
	if err := t.api.AddDomainToSplitDNS(ctx, domain); err != nil {
		return fmt.Errorf("failed to add domain to split DNS: %w", err)
	}

	clog.Infof("Successfully added %s to split DNS", domain)
	return nil
}

// RemoveFromSplitDNS removes the current node's IP from split DNS
func (t *Tailscale) RemoveFromSplitDNS() error {
	if !t.enableSplitDNS {
		return nil
	}

	// Get the correct tailnet name from Tailscale status
	tailnet, err := t.GetTailnetName()
	if err != nil {
		clog.Warningf("Could not auto-detect tailnet: %v, using configured value", err)
	} else {
		// Update the API client with the correct tailnet
		t.api.UpdateTailnet(tailnet)
		clog.Debugf("Updated tailnet to: %s", tailnet)
	}

	domain := t.splitDNSDomain
	clog.Infof("Removing domain %s from split DNS", domain)

	ctx := context.Background()
	if err := t.api.RemoveDomainFromSplitDNS(ctx, domain); err != nil {
		return fmt.Errorf("failed to remove domain from split DNS: %w", err)
	}

	clog.Infof("Successfully removed %s from split DNS", domain)
	return nil
}

// getRefreshInterval returns the refresh interval in seconds from environment variable
// TSC_REFRESH_INTERVAL, defaulting to 30 seconds if not set or invalid.
func getRefreshInterval() time.Duration {
	if intervalStr := os.Getenv("TSC_REFRESH_INTERVAL"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
			return time.Duration(interval) * time.Second
		}
		clog.Warningf("invalid TSC_REFRESH_INTERVAL value '%s', using default 30 seconds", intervalStr)
	}
	return 30 * time.Second
}

// periodicRefresh periodically updates the DNS records.
// Using a ticker allows for better control and cleanup if needed in the future.
func (t *Tailscale) periodicRefresh() {
	interval := getRefreshInterval()
	ticker := time.NewTicker(interval)
	for range ticker.C {
		t.refresh()
	}
}

// refresh fetches the current Tailscale status and updates the local DNS records.
// This ensures that DNS queries reflect the latest network state.
func (t *Tailscale) refresh() {
	ctx := context.Background()
	status, err := t.lc.Status(ctx)
	if err != nil {
		clog.Errorf("failed to get Tailscale status: %v", err)
		return
	}
	if status == nil || status.Self == nil {
		clog.Warning("received nil status or self node from Tailscale")
		return
	}

	newRecords := make(map[string]record)

	// Process self node
	t.processNode(newRecords, status.Self)

	// Process peer nodes
	for _, peer := range status.Peer {
		t.processNode(newRecords, peer)
	}

	t.mu.Lock()
	t.records = newRecords
	t.mu.Unlock()

	// Periodically verify and update split DNS
	t.verifySplitDNS()
}

// verifySplitDNS checks if split DNS is properly configured and updates it if needed
func (t *Tailscale) verifySplitDNS() {
	if !t.enableSplitDNS {
		return
	}

	// Only check every 5 minutes to avoid excessive API calls
	now := time.Now()
	if now.Sub(t.lastSplitDNSCheck) < 5*time.Minute {
		return
	}
	t.lastSplitDNSCheck = now

	// Get current IP
	currentIP, err := t.GetOwnIP()
	if err != nil {
		clog.Errorf("Failed to get own IP for split DNS verification: %v", err)
		return
	}

	// Check if IP has changed or if we need to verify registration
	shouldUpdate := false
	if t.lastVerifiedIP != currentIP {
		clog.Infof("Detected IP change from %s to %s, will verify split DNS", t.lastVerifiedIP, currentIP)
		shouldUpdate = true
	} else if t.lastVerifiedIP == "" {
		// First time verification
		shouldUpdate = true
	}

	// The 5-minute check above already handles periodic verification

	if !shouldUpdate {
		return
	}

	// Get the correct tailnet name from Tailscale status
	tailnet, err := t.GetTailnetName()
	if err != nil {
		clog.Warningf("Could not auto-detect tailnet: %v, using configured value", err)
	} else {
		// Update the API client with the correct tailnet
		t.api.UpdateTailnet(tailnet)
		clog.Debugf("Updated tailnet to: %s", tailnet)
	}

	// Verify domain is in split DNS
	ctx := context.Background()
	isPresent, err := t.api.IsDomainInSplitDNS(ctx, t.splitDNSDomain)
	if err != nil {
		clog.Errorf("Failed to verify split DNS status: %v", err)
		return
	}

	if !isPresent {
		clog.Warningf("Domain %s not found in split DNS, re-adding...", t.splitDNSDomain)
		if err := t.AddToSplitDNS(); err != nil {
			clog.Errorf("Failed to re-add domain to split DNS: %v", err)
			return
		}
	} else {
		clog.Debugf("Split DNS verification successful for domain %s", t.splitDNSDomain)
	}

	// Update last verified IP
	t.lastVerifiedIP = currentIP
}

// processNode adds DNS records for a given node, including any subdomain tags.
// Subdomain tags allow custom DNS mappings for nodes, enhancing flexibility in naming.
func (t *Tailscale) processNode(records map[string]record, peer *ipnstate.PeerStatus) {
	host := strings.ToLower(peer.HostName)
	fqdn := host + "." + t.Domain + "."
	records[fqdn] = t.ipsToRecord(peer.TailscaleIPs)

	if peer.Tags != nil {
		for _, tag := range peer.Tags.AsSlice() {
			if strings.HasPrefix(tag, "tag:subdomain-") {
				sub := strings.TrimPrefix(tag, "tag:subdomain-")
				sub = strings.ReplaceAll(sub, "-", ".")
				subFqdn := host + "." + sub + "." + t.Domain + "."
				records[subFqdn] = t.ipsToRecord(peer.TailscaleIPs)
			}
		}
	}
}

// ipsToRecord converts a list of IP addresses to a record struct,
// selecting the first IPv4 and IPv6 addresses found.
func (t *Tailscale) ipsToRecord(ips []netip.Addr) record {
	var ipv4, ipv6 net.IP
	for _, ip := range ips {
		if ip.Is4() && ipv4 == nil {
			ipv4 = ip.AsSlice()
		} else if ip.Is6() && ipv6 == nil {
			ipv6 = ip.AsSlice()
		}
	}
	return record{IPv4: ipv4, IPv6: ipv6}
}

func (t *Tailscale) Name() string { return "tailscale" }