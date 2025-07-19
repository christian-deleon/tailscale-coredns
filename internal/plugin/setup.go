package plugin

import (
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("tailscale")

// init registers the tailscale plugin with CoreDNS.
func init() {
	plugin.Register("tailscale", setup)
}

// setup configures the Tailscale plugin with the given domains.
func setup(c *caddy.Controller) error {
	var domains []string

	c.Next() // 'tailscale'
	
	// Parse all domains on the same line
	for c.NextArg() {
		domain := c.Val()
		// Split by comma if multiple domains are provided together
		if strings.Contains(domain, ",") {
			parts := strings.Split(domain, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					domains = append(domains, trimmed)
				}
			}
		} else {
			domains = append(domains, domain)
		}
	}

	if len(domains) == 0 {
		return c.ArgErr()
	}

	ts, err := New(domains)
	if err != nil {
		return plugin.Error("tailscale", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		ts.Next = next
		return ts
	})

	return nil
}