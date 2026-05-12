package resolve

import (
	"net/http"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/doh"
	"github.com/coredns/coredns/plugin/pkg/transport"
)

const pluginName = "resolve"

func init() {
	plugin.Register("resolve", setupResolve)
	plugin.Register("edns0", setupEDNS0)
}

func setupResolve(c *caddy.Controller) error {
	conf := dnsserver.GetConfig(c)

	if conf.Transport == transport.HTTPS || conf.Transport == transport.HTTPS3 {
		origValidator := conf.HTTPRequestValidateFunc
		conf.HTTPRequestValidateFunc = func(r *http.Request) bool {
			if r.URL.Path == "/resolve" {
				return true
			}
			if origValidator != nil {
				return origValidator(r)
			}
			return r.URL.Path == doh.Path
		}
	}

	cfg := &Resolve{}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		cfg.Next = next
		return cfg
	})

	return nil
}

func setupEDNS0(c *caddy.Controller) error {
	conf := dnsserver.GetConfig(c)

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) != 1 {
			return c.ArgErr()
		}
		switch args[0] {
		case "on":
			conf.ResolveEDNS0 = true
		case "off":
			conf.ResolveEDNS0 = false
		default:
			return c.ArgErr()
		}
	}

	return nil
}
