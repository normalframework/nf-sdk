package netmgr

import (
	"fmt"
	"net"
	"strings"
)

// PrefixToNetmask renders a CIDR prefix length as a dotted-quad netmask.
func PrefixToNetmask(prefix uint32) string {
	if prefix > 32 {
		return ""
	}
	if prefix == 0 {
		return "0.0.0.0"
	}
	mask := uint32(0xFFFFFFFF) << (32 - prefix)
	return fmt.Sprintf("%d.%d.%d.%d", mask>>24, (mask>>16)&0xFF, (mask>>8)&0xFF, mask&0xFF)
}

// NetmaskToPrefix converts a dotted-quad netmask to a CIDR prefix length. It
// also accepts a bare prefix length ("24"), which is what most people type.
func NetmaskToPrefix(mask string) (uint32, error) {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return 0, fmt.Errorf("empty netmask")
	}
	if !strings.Contains(mask, ".") {
		var n uint32
		if _, err := fmt.Sscanf(mask, "%d", &n); err != nil || n > 32 {
			return 0, fmt.Errorf("invalid prefix length %q", mask)
		}
		return n, nil
	}
	ip := net.ParseIP(mask).To4()
	if ip == nil {
		return 0, fmt.Errorf("invalid netmask %q", mask)
	}
	ones, bits := net.IPv4Mask(ip[0], ip[1], ip[2], ip[3]).Size()
	if bits == 0 {
		return 0, fmt.Errorf("netmask %q is not contiguous", mask)
	}
	return uint32(ones), nil
}

// ipToNMUint32 encodes dotted-quad addresses for NetworkManager's legacy
// "ipv4.dns" property, which is an array of uint32 in host byte order on
// little-endian machines. Carried over from gobac, where this encoding is
// known to work against balenaOS.
func ipToNMUint32(ips ...string) []uint32 {
	var out []uint32
	for _, s := range ips {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		ip := net.ParseIP(s).To4()
		if ip == nil {
			continue
		}
		out = append(out, uint32(ip[0])|uint32(ip[1])<<8|uint32(ip[2])<<16|uint32(ip[3])<<24)
	}
	return out
}

// ParseDNSList splits a comma or space separated list of DNS servers and
// validates each entry.
func ParseDNSList(s string) ([]string, error) {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if net.ParseIP(f) == nil {
			return nil, fmt.Errorf("%q is not a valid IP address", f)
		}
		out = append(out, f)
	}
	return out, nil
}

// percentTodBm approximates a signal percentage as dBm for display: 100% is
// about -30dBm and 0% about -90dBm.
func percentTodBm(pct int32) int32 { return -90 + (pct * 60 / 100) }
