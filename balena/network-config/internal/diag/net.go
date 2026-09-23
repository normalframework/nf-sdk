package diag

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ValidateHost rejects anything that is not a plausible hostname or IP. The
// probes here never invoke a shell, so this is input hygiene rather than an
// injection defence, but it keeps error messages sane. Carried over from
// gobac/src/nwconfig/diagnostics.go.
func ValidateHost(host string) error {
	if host == "" {
		return errors.New("host is required")
	}
	if len(host) > 253 {
		return errors.New("host is too long")
	}
	for _, c := range host {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '-' || c == ':' || c == '_'
		if !ok {
			return fmt.Errorf("invalid character %q in host", c)
		}
	}
	return nil
}

// PingReply is one echo reply, or one timeout.
type PingReply struct {
	Seq     int     `json:"seq"`
	From    string  `json:"from,omitempty"`
	RTTMs   float64 `json:"rttMs,omitempty"`
	Timeout bool    `json:"timeout,omitempty"`
	Error   string  `json:"error,omitempty"`
}

// icmpConn opens an ICMP socket, preferring a raw socket and falling back to
// the unprivileged datagram form. The container gets NET_RAW, but the fallback
// keeps the tool working when it does not.
func icmpConn() (*icmp.PacketConn, bool, error) {
	if c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0"); err == nil {
		return c, false, nil
	}
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return nil, false, fmt.Errorf("could not open an ICMP socket: %w (the container needs NET_RAW, or net.ipv4.ping_group_range must include this user)", err)
	}
	return c, true, nil
}

// Ping sends ICMP echo requests, calling emit for each reply or timeout as it
// happens so the console can stream results rather than waiting for the run.
func Ping(ctx context.Context, host string, count int, emit func(PingReply)) error {
	if err := ValidateHost(host); err != nil {
		return err
	}
	if count < 1 || count > 30 {
		count = 4
	}

	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", host, err)
	}

	conn, datagram, err := icmpConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	// The datagram socket rewrites the echo id, so match on sequence only.
	id := os.Getpid() & 0xffff
	var dst net.Addr = addr
	if datagram {
		dst = &net.UDPAddr{IP: addr.IP}
	}

	for seq := 1; seq <= count; seq++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{ID: id, Seq: seq, Data: []byte("netcfgd-ping")},
		}
		wire, err := msg.Marshal(nil)
		if err != nil {
			return err
		}

		start := time.Now()
		if _, err := conn.WriteTo(wire, dst); err != nil {
			emit(PingReply{Seq: seq, Error: err.Error()})
			continue
		}

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		reply, from, err := readEcho(conn, seq)
		switch {
		case err != nil:
			emit(PingReply{Seq: seq, Timeout: true})
		case reply:
			emit(PingReply{
				Seq:   seq,
				From:  hostOf(from),
				RTTMs: float64(time.Since(start).Microseconds()) / 1000,
			})
		}

		if seq < count {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
		}
	}
	return nil
}

// readEcho waits for an echo reply with the expected sequence, ignoring
// unrelated ICMP traffic that arrives on the shared socket.
func readEcho(conn *icmp.PacketConn, wantSeq int) (bool, net.Addr, error) {
	buf := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return false, nil, err
		}
		msg, err := icmp.ParseMessage(1, buf[:n]) // 1 = IPv4 ICMP
		if err != nil {
			continue
		}
		echo, ok := msg.Body.(*icmp.Echo)
		if msg.Type == ipv4.ICMPTypeEchoReply && ok && echo.Seq == wantSeq {
			return true, peer, nil
		}
	}
}

func hostOf(addr net.Addr) string {
	switch a := addr.(type) {
	case *net.IPAddr:
		return a.IP.String()
	case *net.UDPAddr:
		return a.IP.String()
	}
	return addr.String()
}

// Hop is one traceroute hop.
type Hop struct {
	TTL     int     `json:"ttl"`
	Addr    string  `json:"addr,omitempty"`
	Name    string  `json:"name,omitempty"`
	RTTMs   float64 `json:"rttMs,omitempty"`
	Timeout bool    `json:"timeout,omitempty"`
	Final   bool    `json:"final,omitempty"`
}

// Traceroute walks the path to a host by sending echo requests with an
// increasing TTL, emitting each hop as it is discovered.
func Traceroute(ctx context.Context, host string, maxHops int, emit func(Hop)) error {
	if err := ValidateHost(host); err != nil {
		return err
	}
	if maxHops < 1 || maxHops > 40 {
		maxHops = 20
	}

	addr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", host, err)
	}

	conn, datagram, err := icmpConn()
	if err != nil {
		return err
	}
	defer conn.Close()
	p := conn.IPv4PacketConn()

	id := os.Getpid() & 0xffff
	var dst net.Addr = addr
	if datagram {
		dst = &net.UDPAddr{IP: addr.IP}
	}

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := p.SetTTL(ttl); err != nil {
			return fmt.Errorf("set TTL: %w", err)
		}

		msg := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Body: &icmp.Echo{ID: id, Seq: ttl, Data: []byte("netcfgd-trace")},
		}
		wire, _ := msg.Marshal(nil)

		start := time.Now()
		if _, err := conn.WriteTo(wire, dst); err != nil {
			emit(Hop{TTL: ttl, Timeout: true})
			continue
		}

		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		hopAddr, final, err := readHop(conn)
		if err != nil {
			emit(Hop{TTL: ttl, Timeout: true})
			continue
		}

		hop := Hop{
			TTL:   ttl,
			Addr:  hopAddr,
			RTTMs: float64(time.Since(start).Microseconds()) / 1000,
			Final: final,
		}
		// A reverse name makes a traceroute readable, but never at the cost of
		// stalling the output.
		lookupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		if names, err := net.DefaultResolver.LookupAddr(lookupCtx, hopAddr); err == nil && len(names) > 0 {
			hop.Name = strings.TrimSuffix(names[0], ".")
		}
		cancel()

		emit(hop)
		if final {
			return nil
		}
	}
	return nil
}

// readHop returns the responding address and whether it is the destination.
func readHop(conn *icmp.PacketConn) (string, bool, error) {
	buf := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return "", false, err
		}
		msg, err := icmp.ParseMessage(1, buf[:n])
		if err != nil {
			continue
		}
		switch msg.Type {
		case ipv4.ICMPTypeTimeExceeded:
			return hostOf(peer), false, nil
		case ipv4.ICMPTypeEchoReply:
			return hostOf(peer), true, nil
		case ipv4.ICMPTypeDestinationUnreachable:
			return hostOf(peer), true, nil
		}
	}
}

// DNSResult reports a name resolution attempt.
type DNSResult struct {
	Host    string   `json:"host"`
	Addrs   []string `json:"addrs,omitempty"`
	Millis  int64    `json:"millis"`
	Servers []string `json:"servers,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// ResolveDNS times a lookup and reports the resolvers in use.
func ResolveDNS(ctx context.Context, host string) DNSResult {
	res := DNSResult{Host: host, Servers: Resolvers()}
	if err := ValidateHost(host); err != nil {
		res.Error = err.Error()
		return res
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	res.Millis = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Addrs = addrs
	return res
}

// Resolvers reads the nameservers this process will use.
func Resolvers() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			out = append(out, fields[1])
		}
	}
	return out
}

// TCPCheck reports whether a host:port accepts a connection.
func TCPCheck(ctx context.Context, addr string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return fmt.Sprintf("connected to %s in %dms", conn.RemoteAddr(), time.Since(start).Milliseconds()), nil
}

// DefaultGateway returns the current IPv4 default gateway from the kernel
// routing table. Reading /proc/net/route avoids depending on iproute2 being
// installed in the image.
func DefaultGateway() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 || f[1] != "00000000" {
			continue
		}
		if ip := parseHexIPLE(f[2]); ip != "" {
			return ip
		}
	}
	return ""
}

// parseHexIPLE decodes the little-endian hex addresses in /proc/net/route,
// where the leftmost byte pair is the least significant byte of the address.
func parseHexIPLE(h string) string {
	if len(h) != 8 {
		return ""
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return ""
	}
	return net.IPv4(byte(v), byte(v>>8), byte(v>>16), byte(v>>24)).String()
}
