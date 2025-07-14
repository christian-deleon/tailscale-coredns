package tailscale

import (
	"context"
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
}

func New(domain string) (*Tailscale, error) {
	ts := &Tailscale{
		Domain:  domain,
		records: make(map[string]record),
		lc:      &tailscale.LocalClient{Socket: "/run/tailscale/tailscaled.sock"},
	}
	
	go ts.periodicRefresh()
	return ts, nil
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