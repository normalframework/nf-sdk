package diag

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TunnelStatus reports on the Cloudflare tunnel that provides remote access.
type TunnelStatus struct {
	// Registered is true when cloudflared has successfully registered with the
	// edge at least once.
	Registered bool `json:"registered"`
	// Connections is the number of active edge connections cloudflared holds.
	Connections int `json:"connections"`
	// MetricsError explains why the local cloudflared metrics could not be
	// read, which usually just means cloudflared is not running here.
	MetricsError string `json:"metricsError,omitempty"`

	// QUICViable reports whether UDP/7844 reaches the Cloudflare edge.
	QUICViable bool   `json:"quicViable"`
	QUICDetail string `json:"quicDetail"`

	// Advice is the recommended action, in the terms an operator needs.
	Advice string `json:"advice,omitempty"`
}

// metricsAddress matches the address gobac's platform service uses, so both
// read the same cloudflared instance.
func metricsAddress() string {
	if a := os.Getenv("PROMETHEUS_CLOUDFLARED_ADDRESS"); a != "" {
		return a
	}
	return "http://127.0.0.1:10005"
}

// readTunnelMetrics scrapes cloudflared's Prometheus endpoint. This is the same
// signal gobac/src/platform/services.go uses to report tunnel health.
func readTunnelMetrics(ctx context.Context, st *TunnelStatus) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsAddress()+"/metrics", nil)
	if err != nil {
		st.MetricsError = err.Error()
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		st.MetricsError = fmt.Sprintf("cloudflared metrics unavailable at %s: %v", metricsAddress(), err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "cloudflared_tunnel_tunnel_register_success"):
			if v, ok := lastValue(line); ok && v > 0 {
				st.Registered = true
			}
		case strings.HasPrefix(line, "cloudflared_tunnel_ha_connections"):
			if v, ok := lastValue(line); ok {
				st.Connections = int(v)
			}
		}
	}
}

// lastValue pulls the sample value off the end of a Prometheus text line.
func lastValue(line string) (float64, bool) {
	i := strings.LastIndex(line, " ")
	if i < 0 {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(line[i+1:]), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// CheckTunnel reports whether remote access can work on this network.
//
// proxyConfigured tells the check whether a host proxy is in force, which
// changes the advice completely: redsocks only redirects TCP, so a configured
// proxy means QUIC cannot reach the edge no matter what the UDP probe says.
func CheckTunnel(ctx context.Context, proxyConfigured bool, tcpReachable bool) TunnelStatus {
	var st TunnelStatus
	readTunnelMetrics(ctx, &st)

	st.QUICViable, st.QUICDetail = probeQUIC(ctx, "region1.v2.argotunnel.com:7844")

	switch {
	case proxyConfigured && !tcpReachable && st.QUICViable:
		// The interesting middle case. cloudflared prefers QUIC over UDP,
		// which redsocks never redirects, so it leaves the device directly and
		// works — the tunnel is up despite the proxy refusing TCP/7844. That
		// is fine until someone tightens UDP, at which point remote access
		// disappears with no other change on the device.
		st.Advice = "The proxy refuses TCP/7844, but the tunnel works anyway because QUIC goes out over UDP and bypasses the proxy. That is fragile — block UDP/7844 and remote access stops. Ask for CONNECT to region1 and region2.v2.argotunnel.com:7844."
	case proxyConfigured && !tcpReachable:
		st.Advice = "No path to the tunnel edge: the proxy refuses TCP/7844 and UDP/7844 is blocked. The port cannot be moved to 443. Remote access needs CONNECT to region1 and region2.v2.argotunnel.com:7844."
	case proxyConfigured && !st.QUICViable:
		st.Advice = "The proxy carries TCP only, so QUIC cannot be used. TCP to the edge works — pin protocol: http2 in tunnel.yaml to skip the QUIC timeout on every attempt."
	case !tcpReachable && !st.QUICViable:
		st.Advice = "Port 7844 is unreachable over both TCP and UDP. Ask for outbound 7844 to region1 and region2.v2.argotunnel.com."
	case !st.QUICViable && tcpReachable:
		st.Advice = "UDP/7844 is blocked, TCP works. cloudflared falls back on its own, but pinning protocol: http2 avoids the QUIC timeout first."
	}
	return st
}

// probeQUIC tests whether UDP reaches the Cloudflare edge, using QUIC's own
// version negotiation as the signal.
//
// A QUIC server that receives a long-header packet carrying a version it does
// not support must reply with a Version Negotiation packet (RFC 9000 section
// 6). Sending a deliberately unsupported version is therefore a reliable,
// protocol-legal way to get a response without completing a handshake — much
// better than sending UDP into the void and guessing from silence.
func probeQUIC(ctx context.Context, addr string) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		return false, fmt.Sprintf("could not open a UDP socket to %s: %v", addr, err)
	}
	defer conn.Close()

	pkt, dcid, err := versionNegotiationPacket()
	if err != nil {
		return false, err.Error()
	}
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	if _, err := conn.Write(pkt); err != nil {
		return false, fmt.Sprintf("could not send to %s: %v", addr, err)
	}

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return false, fmt.Sprintf("no UDP reply from %s within 5s — UDP/7844 is most likely blocked", addr)
	}
	// A Version Negotiation packet has the long-header bit set and a zero
	// version field.
	if n >= 7 && buf[0]&0x80 != 0 && buf[1] == 0 && buf[2] == 0 && buf[3] == 0 && buf[4] == 0 {
		// The server echoes our source connection ID back as its destination.
		_ = dcid
		return true, fmt.Sprintf("QUIC version negotiation received from %s (%d bytes) — UDP/7844 is open", addr, n)
	}
	return true, fmt.Sprintf("received %d bytes of UDP from %s — the path is open, though the reply was not version negotiation", n, addr)
}

// versionNegotiationPacket builds a QUIC long-header Initial packet carrying a
// reserved version, padded to the 1200-byte minimum that RFC 9000 requires of
// Initial packets so middleboxes and the server accept it.
func versionNegotiationPacket() ([]byte, []byte, error) {
	dcid := make([]byte, 8)
	scid := make([]byte, 8)
	if _, err := rand.Read(dcid); err != nil {
		return nil, nil, fmt.Errorf("generate connection id: %w", err)
	}
	if _, err := rand.Read(scid); err != nil {
		return nil, nil, fmt.Errorf("generate connection id: %w", err)
	}

	pkt := []byte{0xC3} // long header, Initial, 4-byte packet number
	// A version with the 0x?a?a?a?a pattern is reserved to force version
	// negotiation rather than being mistaken for a real QUIC version.
	pkt = append(pkt, 0x0a, 0x0a, 0x0a, 0x0a)
	pkt = append(pkt, byte(len(dcid)))
	pkt = append(pkt, dcid...)
	pkt = append(pkt, byte(len(scid)))
	pkt = append(pkt, scid...)
	pkt = append(pkt, 0x00) // zero-length token
	for len(pkt) < 1200 {
		pkt = append(pkt, 0x00)
	}
	return pkt, dcid, nil
}
