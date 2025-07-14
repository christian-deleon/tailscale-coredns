package tailscale

import (
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

// setup configures the Tailscale plugin with the given domain.
func setup(c *caddy.Controller) error {
	var domain string
	
	c.Next() // 'tailscale'
	if c.NextArg() {
		domain = c.Val()
	} else {
		return c.ArgErr()
	}
	
	// Check for any additional arguments
	if c.NextArg() {
		return c.ArgErr()
	}

	ts, err := New(domain)
	if err != nil {
		return plugin.Error("tailscale", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		ts.Next = next
		return ts
	})

	return nil
}