package tailscale

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("tailscale")

func init() {
	plugin.Register("tailscale", setup)
}

func setup(c *caddy.Controller) error {
	var domain string
	c.Next() // 'tailscale'
	if c.NextArg() {
		domain = c.Val()
	} else {
		return c.ArgErr()
	}
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