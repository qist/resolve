package resolve

import (
	"context"

	"github.com/coredns/coredns/plugin"

	"github.com/miekg/dns"
)

type Resolve struct {
	Next plugin.Handler
}

func (r *Resolve) Name() string { return pluginName }

func (r *Resolve) ServeDNS(ctx context.Context, w dns.ResponseWriter, msg *dns.Msg) (int, error) {
	return plugin.NextOrFailure(r.Name(), r.Next, ctx, w, msg)
}
