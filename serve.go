package tailscale

import (
	"context"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

func (t *Tailscale) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	name := state.Name()
	if !strings.HasSuffix(name, t.Domain+".") {
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}

	t.mu.RLock()
	rec, ok := t.records[name]
	t.mu.RUnlock()
	if !ok {
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	header := dns.RR_Header{Name: name, Rrtype: state.QType(), Class: state.QClass(), Ttl: 60}

	switch state.QType() {
	case dns.TypeA:
		if rec.ipv4 == nil {
			return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
		}
		m.Answer = append(m.Answer, &dns.A{Hdr: header, A: rec.ipv4})
	case dns.TypeAAAA:
		if rec.ipv6 == nil {
			return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
		}
		m.Answer = append(m.Answer, &dns.AAAA{Hdr: header, AAAA: rec.ipv6})
	default:
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}

	if err := w.WriteMsg(m); err != nil {
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}

func (t *Tailscale) Name() string { return "tailscale" }