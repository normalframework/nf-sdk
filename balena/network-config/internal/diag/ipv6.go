package diag

import (
	"net"
	"os"
	"strings"
)

// IPv6Bypass reports whether this device has global IPv6 connectivity.
//
// It matters only when a proxy is configured, and then it matters a great deal.
// balenaOS's transparent proxy is built from iptables rules and has no
// ip6tables equivalent, so IPv6 traffic is never redirected: on a dual-stack
// network some traffic goes through the proxy and some does not, decided by
// whichever address family a name happens to resolve to first.
//
// The practical result is a device that appears to work while its proxy is
// entirely broken, and an audit trail on the proxy that is missing half the
// connections.
type IPv6Bypass struct {
	HasGlobalIPv6 bool     `json:"hasGlobalIpv6"`
	Addresses     []string `json:"addresses,omitempty"`
	Advice        string   `json:"advice,omitempty"`
}

// CheckIPv6Bypass looks for global IPv6 addresses on the host's interfaces.
func CheckIPv6Bypass(proxyConfigured bool) IPv6Bypass {
	var out IPv6Bypass

	for _, name := range hostInterfaces() {
		iface, err := net.InterfaceByName(name)
		if err != nil {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() != nil {
				continue
			}
			// Link-local and loopback never carry traffic off the device.
			if ipnet.IP.IsLinkLocalUnicast() || ipnet.IP.IsLoopback() {
				continue
			}
			out.HasGlobalIPv6 = true
			out.Addresses = append(out.Addresses, name+" "+ipnet.IP.String())
		}
	}

	if out.HasGlobalIPv6 && proxyConfigured {
		out.Advice = "A proxy is configured but this device has global IPv6. balenaOS redirects " +
			"IPv4 only, so anything with a AAAA record bypasses the proxy entirely and keeps " +
			"working even if the proxy is down. Disable IPv6 if all traffic must be proxied."
	}
	return out
}

// hostInterfaces lists interface names from sysfs, which reflects the host's
// network namespace when the container runs with host networking.
func hostInterfaces() []string {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if name == "lo" || strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "balena") ||
			strings.HasPrefix(name, "resin-") || strings.HasPrefix(name, "docker") {
			continue
		}
		out = append(out, name)
	}
	return out
}
