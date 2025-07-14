package tailscale

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"net/netip"

	"tailscale.com/client/tailscale"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

type Tailscale struct {
	Next       plugin.Handler
	Domain     string
	records    map[string]struct{ ipv4, ipv6 net.IP }
	mu         sync.RWMutex
	lc         *tailscale.LocalClient
}

func New(domain string) (*Tailscale, error) {
	ts := &Tailscale{
		Domain:  domain,
		records: make(map[string]struct{ ipv4, ipv6 net.IP }),
		lc:      &tailscale.LocalClient{Socket: "/run/tailscale/tailscaled.sock"},
	}
	go ts.periodicRefresh()
	return ts, nil
}

func (t *Tailscale) periodicRefresh() {
	for {
		t.refresh()
		time.Sleep(30 * time.Second)
	}
}

func (t *Tailscale) refresh() {
	ctx := context.Background()
	status, err := t.lc.Status(ctx)
	if err != nil {
		clog.Error(err)
		return
	}
	if status == nil || status.Self == nil {
		return
	}

	newRecords := make(map[string]struct{ ipv4, ipv6 net.IP })

	addRecord := func(fqdn string, ips []netip.Addr) {
		var ipv4, ipv6 net.IP
		for _, ip := range ips {
			if ip.Is4() {
				ipv4 = ip.AsSlice()
			} else if ip.Is6() {
				ipv6 = ip.AsSlice()
			}
		}
		newRecords[fqdn] = struct{ ipv4, ipv6 net.IP }{ipv4, ipv6}
	}

	// Self
	self := status.Self
	host := strings.ToLower(self.HostName)
	fqdn := host + "." + t.Domain + "."
	addRecord(fqdn, self.TailscaleIPs)
	if self.Tags != nil {
		for _, tag := range self.Tags.AsSlice() {
			if strings.HasPrefix(tag, "tag:subdomain-") {
				sub := strings.TrimPrefix(tag, "tag:subdomain-")
				sub = strings.ReplaceAll(sub, "-", ".")
				subFqdn := host + "." + sub + "." + t.Domain + "."
				addRecord(subFqdn, self.TailscaleIPs)
			}
		}
	}

	// Peers
	if status.Peer != nil {
		for _, peer := range status.Peer {
			host = strings.ToLower(peer.HostName)
			fqdn = host + "." + t.Domain + "."
			addRecord(fqdn, peer.TailscaleIPs)
			if peer.Tags != nil {
				for _, tag := range peer.Tags.AsSlice() {
					if strings.HasPrefix(tag, "tag:subdomain-") {
						sub := strings.TrimPrefix(tag, "tag:subdomain-")
						sub = strings.ReplaceAll(sub, "-", ".")
						subFqdn := host + "." + sub + "." + t.Domain + "."
						addRecord(subFqdn, peer.TailscaleIPs)
					}
				}
			}
		}
	}

	t.mu.Lock()
	t.records = newRecords
	t.mu.Unlock()
}