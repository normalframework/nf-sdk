// Package diag runs network diagnostics for the console: reachability checks
// against the endpoints this device actually depends on, plus ICMP and DNS
// tools for working out why one of them fails.
package diag

import (
	"context"
	"sync"
	"time"

	"github.com/normalframework/netcfgd/internal/proxyprobe"
)

// Group orders the checklist in the console.
type Group string

const (
	GroupBalena Group = "balena"
	GroupNF     Group = "normal framework"
	GroupTunnel Group = "cloudflare tunnel"
	GroupBasic  Group = "internet"
)

// Endpoint is one entry in the reachability checklist.
type Endpoint struct {
	proxyprobe.Target
	Group    Group `json:"group"`
	Required bool  `json:"required"`
}

// Endpoints is the list of destinations this device needs, in the order the
// console shows them. This doubles as the document to hand a customer's IT
// department when asking for firewall or proxy allowances.
func Endpoints() []Endpoint {
	return []Endpoint{
		{Group: GroupBasic, Required: true, Target: proxyprobe.Target{
			Name: "Generic HTTPS", Addr: "cloudflare.com:443", Kind: proxyprobe.KindHTTPS,
			Note: "Baseline check. If this fails, nothing else will work.",
		}},

		{Group: GroupBalena, Required: true, Target: proxyprobe.Target{
			Name: "balena API", Addr: "api.balena-cloud.com:443", Kind: proxyprobe.KindHTTPS, Path: "/ping",
			Note: "Device management. Without it the device cannot report state or receive updates.",
		}},
		{Group: GroupBalena, Required: true, Target: proxyprobe.Target{
			Name: "balena registry", Addr: "registry2.balena-cloud.com:443", Kind: proxyprobe.KindHTTPS,
			Note: "Container image pulls. Required for any software update.",
		}},
		{Group: GroupBalena, Required: true, Target: proxyprobe.Target{
			// balena renamed this service from vpn to cloudlink;
			// vpn.balena-cloud.com no longer resolves.
			// This is an OpenVPN endpoint, not an HTTPS server: it does not
			// complete a TLS handshake with an ordinary client, so only
			// reachability is meaningful here.
			Name: "balena cloudlink", Addr: "cloudlink.balena-cloud.com:443", Kind: proxyprobe.KindTCP,
			Note: "Device VPN, used for remote access and the web terminal from balenaCloud.",
		}},
		{Group: GroupBalena, Required: false, Target: proxyprobe.Target{
			Name: "balena delta", Addr: "delta.balena-cloud.com:443", Kind: proxyprobe.KindHTTPS,
			Note: "Differential updates. Updates still work without it, just more slowly.",
		}},

		{Group: GroupNF, Required: true, Target: proxyprobe.Target{
			Name: "NF portal", Addr: "portal.normal-online.net:443", Kind: proxyprobe.KindHTTPS,
			Note: "Normal Framework cloud portal and licensing.",
		}},
		{Group: GroupNF, Required: false, Target: proxyprobe.Target{
			Name: "NF container registry", Addr: "normalframework.azurecr.io:443", Kind: proxyprobe.KindHTTPS,
			Note: "Normal Framework image pulls.",
		}},

		// The tunnel edge is the check that most often surprises people. It
		// runs on TCP/7844 rather than 443, and Cloudflare does not allow the
		// port to be changed, so a proxy that only permits CONNECT to 443
		// cannot carry remote access at all.
		{Group: GroupTunnel, Required: true, Target: proxyprobe.Target{
			Name: "Tunnel edge (region1)", Addr: "region1.v2.argotunnel.com:7844", Kind: proxyprobe.KindReachable,
			Note: "Remote access. Port 7844 is fixed by Cloudflare and cannot be moved to 443.",
		}},
		{Group: GroupTunnel, Required: true, Target: proxyprobe.Target{
			Name: "Tunnel edge (region2)", Addr: "region2.v2.argotunnel.com:7844", Kind: proxyprobe.KindReachable,
			Note: "Second tunnel region; cloudflared uses both.",
		}},
	}
}

// EndpointResult pairs an endpoint with its probe outcome.
type EndpointResult struct {
	Endpoint Endpoint          `json:"endpoint"`
	Result   proxyprobe.Result `json:"result"`
}

// CheckEndpoints probes every endpoint concurrently.
//
// Pass the host's proxy when one is configured, rather than nil. A nil proxy
// relies on redsocks intercepting transparently, and that hides exactly the
// failure worth seeing: the connection which succeeds is the one to redsocks,
// so a destination the proxy goes on to refuse still looks reachable. Probing
// through the proxy explicitly attributes the refusal to the connect stage,
// where it belongs.
func CheckEndpoints(ctx context.Context, p *proxyprobe.Proxy, endpoints []Endpoint, timeout time.Duration) []EndpointResult {
	out := make([]EndpointResult, len(endpoints))
	var wg sync.WaitGroup
	// Bound concurrency: these run on gateways with very little CPU, and a
	// stampede of TLS handshakes measurably skews the timings we report.
	sem := make(chan struct{}, 4)

	for i, ep := range endpoints {
		wg.Add(1)
		go func(i int, ep Endpoint) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = EndpointResult{
				Endpoint: ep,
				Result:   proxyprobe.Probe(ctx, p, ep.Target, timeout),
			}
		}(i, ep)
	}
	wg.Wait()
	return out
}
