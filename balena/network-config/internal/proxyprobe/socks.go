package proxyprobe

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

// socks5 reply codes, RFC 1928 section 6.
var socks5Errors = map[byte]string{
	0x01: "general SOCKS server failure",
	0x02: "connection not allowed by ruleset",
	0x03: "network unreachable",
	0x04: "host unreachable",
	0x05: "connection refused by destination",
	0x06: "TTL expired",
	0x07: "command not supported",
	0x08: "address type not supported",
}

// socks5Greet performs the SOCKS5 method negotiation (RFC 1928 section 3) and
// reports which authentication method the proxy chose.
func socks5Greet(conn net.Conn, haveCreds bool) (byte, error) {
	methods := []byte{0x00} // no authentication
	if haveCreds {
		methods = []byte{0x02, 0x00} // prefer username/password
	}
	req := append([]byte{0x05, byte(len(methods))}, methods...)
	if _, err := conn.Write(req); err != nil {
		return 0, fmt.Errorf("send greeting: %w", err)
	}

	var resp [2]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return 0, fmt.Errorf("read greeting reply: %w", err)
	}
	if resp[0] != 0x05 {
		return 0, fmt.Errorf("not a SOCKS5 proxy (version byte 0x%02x)", resp[0])
	}
	if resp[1] == 0xFF {
		return 0, fmt.Errorf("proxy rejected all offered authentication methods")
	}
	return resp[1], nil
}

// socks5Auth performs username/password authentication, RFC 1929.
func socks5Auth(conn net.Conn, user, pass string) error {
	if len(user) > 255 || len(pass) > 255 {
		return fmt.Errorf("username and password must each be 255 bytes or fewer")
	}
	buf := []byte{0x01, byte(len(user))}
	buf = append(buf, user...)
	buf = append(buf, byte(len(pass)))
	buf = append(buf, pass...)
	if _, err := conn.Write(buf); err != nil {
		return fmt.Errorf("send credentials: %w", err)
	}

	var resp [2]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return fmt.Errorf("read auth reply: %w", err)
	}
	if resp[1] != 0x00 {
		return fmt.Errorf("proxy rejected the credentials (status 0x%02x)", resp[1])
	}
	return nil
}

// socks5Connect issues a CONNECT for the target host and port.
func socks5Connect(conn net.Conn, host string, port int) error {
	req := []byte{0x05, 0x01, 0x00}
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			req = append(req, 0x01)
			req = append(req, v4...)
		} else {
			req = append(req, 0x04)
			req = append(req, ip.To16()...)
		}
	} else {
		if len(host) > 255 {
			return fmt.Errorf("hostname is too long for SOCKS5")
		}
		req = append(req, 0x03, byte(len(host)))
		req = append(req, host...)
	}
	req = binary.BigEndian.AppendUint16(req, uint16(port))

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("send connect request: %w", err)
	}

	head := make([]byte, 4)
	if _, err := io.ReadFull(conn, head); err != nil {
		return fmt.Errorf("read connect reply: %w", err)
	}
	if head[1] != 0x00 {
		if msg, ok := socks5Errors[head[1]]; ok {
			return fmt.Errorf("proxy refused: %s", msg)
		}
		return fmt.Errorf("proxy refused with status 0x%02x", head[1])
	}

	// Drain the bound address so the connection is left at the start of the
	// tunnelled stream.
	var skip int
	switch head[3] {
	case 0x01:
		skip = 4
	case 0x04:
		skip = 16
	case 0x03:
		var l [1]byte
		if _, err := io.ReadFull(conn, l[:]); err != nil {
			return fmt.Errorf("read bound address length: %w", err)
		}
		skip = int(l[0])
	default:
		return fmt.Errorf("proxy returned an unknown address type 0x%02x", head[3])
	}
	if _, err := io.CopyN(io.Discard, conn, int64(skip)+2); err != nil {
		return fmt.Errorf("read bound address: %w", err)
	}
	return nil
}

var socks4Errors = map[byte]string{
	0x5B: "request rejected or failed",
	0x5C: "request rejected: proxy could not reach identd on this host",
	0x5D: "request rejected: identd could not confirm the user id",
}

// socks4Connect performs a SOCKS4/4a CONNECT. SOCKS4 has no password
// authentication; the user field is an identd user id, which is why a login
// configured for a socks4 proxy is reported as ignored rather than sent.
func socks4Connect(conn net.Conn, host string, port int, user string) error {
	req := []byte{0x04, 0x01}
	req = binary.BigEndian.AppendUint16(req, uint16(port))

	var domain string
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		req = append(req, ip.To4()...)
	} else {
		// SOCKS4a: an address of 0.0.0.x signals a trailing hostname.
		req = append(req, 0x00, 0x00, 0x00, 0x01)
		domain = host
	}
	req = append(req, user...)
	req = append(req, 0x00)
	if domain != "" {
		req = append(req, domain...)
		req = append(req, 0x00)
	}

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("send connect request: %w", err)
	}

	resp := make([]byte, 8)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("read connect reply: %w", err)
	}
	if resp[0] != 0x00 {
		return fmt.Errorf("not a SOCKS4 proxy (reply version 0x%02x)", resp[0])
	}
	if resp[1] != 0x5A {
		if msg, ok := socks4Errors[resp[1]]; ok {
			return fmt.Errorf("proxy refused: %s", msg)
		}
		return fmt.Errorf("proxy refused with status 0x%02x", resp[1])
	}
	return nil
}

// splitHostPort separates a target into host and numeric port.
func splitHostPort(target string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, fmt.Errorf("target %q must be host:port", target)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("target %q has an invalid port", target)
	}
	return host, port, nil
}
