package proxyprobe

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Proxy describes a proxy to test. It mirrors the Supervisor's proxy fields but
// stays independent of that package so probes can be exercised in tests without
// a balena device.
type Proxy struct {
	Type     string
	IP       string
	Port     int
	Login    string
	Password string
}

func (p Proxy) addr() string { return net.JoinHostPort(p.IP, fmt.Sprint(p.Port)) }

// Kind selects how far a probe goes once the connection is open.
type Kind string

const (
	// KindHTTPS completes TLS and fetches a URL. Use for real endpoints.
	KindHTTPS Kind = "https"
	// KindTLS completes TLS only. Use where an HTTP request is not meaningful.
	KindTLS Kind = "tls"
	// KindTCP stops once the connection is open.
	//
	// Be careful with this one behind a transparent redirector such as
	// redsocks: the connection that succeeds is the one to the redirector, so
	// a destination the proxy later refuses still looks reachable. Prefer
	// KindReachable, or probe explicitly through the proxy.
	KindTCP Kind = "tcp"
	// KindReachable completes a TLS handshake but does not verify the
	// certificate chain. It answers "do bytes reach this service end to end",
	// which is the only question worth asking of endpoints that present
	// certificates never meant for public validation — the Cloudflare Tunnel
	// edge on 7844 serves a CloudFlare Origin certificate, and cloudflared
	// pins its own CA rather than using the system trust store.
	//
	// Identity is deliberately not checked here. Use KindTLS or KindHTTPS
	// anywhere that matters.
	KindReachable Kind = "reachable"
)

// Target is something to reach.
type Target struct {
	Name string `json:"name"`
	Addr string `json:"addr"` // host:port
	Kind Kind   `json:"kind"`
	Path string `json:"path,omitempty"` // request path for KindHTTPS
	Note string `json:"note,omitempty"` // why this endpoint matters

	// Network selects the address family: "tcp" (default), "tcp4" or "tcp6".
	//
	// This is not a detail. balenaOS's transparent proxy redirects IPv4 only —
	// there are no ip6tables rules — so on a dual-stack network IPv6 traffic
	// bypasses the proxy completely. A probe that is meant to measure the path
	// the proxy governs must therefore ask for IPv4, or it will happily report
	// success over IPv6 while the proxy is dead.
	Network string `json:"network,omitempty"`
}

// network returns the dial network for this target.
func (t Target) network() string {
	if t.Network == "" {
		return "tcp"
	}
	return t.Network
}

// DefaultTimeout bounds a single probe. Field networks are slow and a proxy
// that is silently dropping traffic takes the full timeout to reveal itself,
// so this is a compromise between honesty and a responsive console.
const DefaultTimeout = 10 * time.Second

// Probe runs one connection attempt and reports every stage. A nil proxy tests
// the direct path.
func Probe(ctx context.Context, p *Proxy, t Target, timeout time.Duration) Result {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	via := "direct"
	if p != nil {
		via = fmt.Sprintf("%s proxy %s", p.Type, p.addr())
	}

	rec := newRecorder()
	var interception *Interception

	targetHost, targetPort, err := splitHostPort(t.Addr)
	if err != nil {
		rec.run(StageDNS, func() (string, error) { return "", err })
		return rec.result(t.Addr, via)
	}

	// Resolve whichever host we are about to dial. Separating this from the
	// TCP stage is what distinguishes "DNS is broken on this network" from
	// "the proxy is unreachable", which look the same to most tools.
	dialHost := targetHost
	if p != nil {
		dialHost = p.IP
	}
	var addrs []string
	rec.run(StageDNS, func() (string, error) {
		if ip := net.ParseIP(dialHost); ip != nil {
			addrs = []string{dialHost}
			return dialHost + " (literal address)", nil
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, ipNetwork(t.network()), dialHost)
		if err != nil {
			return "", err
		}
		for _, ip := range ips {
			addrs = append(addrs, ip.String())
		}
		return fmt.Sprintf("%s -> %s", dialHost, strings.Join(addrs, ", ")), nil
	})

	dialAddr := t.Addr
	if p != nil {
		dialAddr = p.addr()
	}

	var conn net.Conn
	ok := rec.run(StageTCP, func() (string, error) {
		d := net.Dialer{}
		c, err := d.DialContext(ctx, t.network(), dialAddr)
		if err != nil {
			return "", err
		}
		conn = c
		return fmt.Sprintf("connected to %s", c.RemoteAddr()), nil
	})
	if !ok {
		return rec.result(t.Addr, via).withInterception(interception)
	}
	defer conn.Close()
	if dl, hasDL := ctx.Deadline(); hasDL {
		conn.SetDeadline(dl)
	}

	// buffered holds any bytes the proxy handshake read ahead of the tunnel.
	var buffered *bufio.Reader
	relay := false

	if p == nil {
		rec.skip(StageHandshake, "no proxy: connected directly")
		rec.skip(StageAuth, "no proxy")
		rec.skip(StageConnect, "no proxy")
	} else {
		buffered, relay = proxyHandshake(rec, conn, p, targetHost, targetPort, t)
		if rec.failed != "" {
			return rec.result(t.Addr, via).withInterception(interception)
		}
	}

	// An http-relay proxy does not tunnel; it forwards whole HTTP requests, so
	// TLS cannot be layered over it. Saying that plainly here saves a long
	// debugging session, because the failure otherwise looks like a broken
	// certificate.
	if relay {
		rec.skip(StageTLS, "http-relay proxies forward HTTP requests and cannot carry TLS")
		rec.run(StageHTTP, func() (string, error) { return readRelayResponse(buffered) })
		return rec.result(t.Addr, via).withInterception(interception)
	}

	var tlsConn *tls.Conn
	switch t.Kind {
	case KindTCP:
		rec.skip(StageTLS, "plain TCP check")
		rec.skip(StageHTTP, "plain TCP check")
		return rec.result(t.Addr, via).withInterception(interception)
	default:
		// Reachability checks deliberately do not assert identity; see
		// KindReachable.
		checkIdentity := t.Kind != KindReachable

		rec.run(StageTLS, func() (string, error) {
			// The handshake defers verification so the presented chain can be
			// inspected even when it does not validate — that is what lets a
			// re-encrypting proxy be named rather than just reported as a
			// certificate error. verifyChain below is what actually enforces
			// trust, and it runs for every probe that asserts identity.
			tlsConn = tls.Client(conn, &tls.Config{
				ServerName:         targetHost,
				InsecureSkipVerify: true, //nolint:gosec // verified explicitly below
			})
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				return "", err
			}
			st := tlsConn.ConnectionState()

			if !checkIdentity {
				issuer := "unknown"
				if len(st.PeerCertificates) > 0 {
					issuer = st.PeerCertificates[0].Issuer.CommonName
				}
				return fmt.Sprintf("%s, reached the service (issuer %q, identity not checked)",
					tlsVersion(st.Version), issuer), nil
			}

			err := verifyChain(st, targetHost)
			if err != nil && untrustedAuthority(err) {
				// Someone is re-issuing certificates for this destination.
				// Record who, so the CA can be recognised and trusted.
				interception = describeInterception(st, false)
				return "", fmt.Errorf("TLS is being intercepted by %q: %w",
					interception.LeafIssuer, err)
			}
			if err != nil {
				return "", err
			}

			// The chain validated. If it validated against a CA we were given
			// rather than a public root, interception is present and working —
			// worth saying, because it explains the issuer name.
			if len(st.PeerCertificates) > 0 && !publiclyRooted(st) {
				interception = describeInterception(st, true)
			}
			return fmt.Sprintf("%s, issuer %q", tlsVersion(st.Version),
				st.PeerCertificates[0].Issuer.CommonName), nil
		})
	}
	if rec.failed != "" {
		return rec.result(t.Addr, via).withInterception(interception)
	}

	if t.Kind == KindTLS || t.Kind == KindReachable {
		rec.skip(StageHTTP, "reachability check, no request sent")
		return rec.result(t.Addr, via).withInterception(interception)
	}

	rec.run(StageHTTP, func() (string, error) {
		path := t.Path
		if path == "" {
			path = "/"
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodHead,
			"https://"+t.Addr+path, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "netcfgd")
		if err := req.Write(tlsConn); err != nil {
			return "", fmt.Errorf("send request: %w", err)
		}
		resp, err := http.ReadResponse(bufio.NewReader(tlsConn), req)
		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}
		defer resp.Body.Close()
		// Any HTTP response proves end-to-end reachability. A 4xx from an API
		// endpoint we are not authorised to call still means the path works.
		return resp.Status, nil
	})

	return rec.result(t.Addr, via).withInterception(interception)
}

// proxyHandshake runs the proxy-specific stages. It returns any buffered reader
// left over from the handshake and whether the proxy is a relay.
func proxyHandshake(rec *recorder, conn net.Conn, p *Proxy, host string, port int, t Target) (*bufio.Reader, bool) {
	switch strings.ToLower(p.Type) {
	case "socks5":
		var method byte
		rec.run(StageHandshake, func() (string, error) {
			m, err := socks5Greet(conn, p.Login != "" || p.Password != "")
			method = m
			if err != nil {
				return "", err
			}
			if m == 0x02 {
				return "SOCKS5, username/password required", nil
			}
			return "SOCKS5, no authentication required", nil
		})
		if rec.failed != "" {
			return nil, false
		}

		if method == 0x02 {
			rec.run(StageAuth, func() (string, error) {
				if p.Login == "" && p.Password == "" {
					return "", fmt.Errorf("proxy requires a username and password but none are configured")
				}
				return "accepted", socks5Auth(conn, p.Login, p.Password)
			})
		} else if p.Login != "" {
			rec.skip(StageAuth, "proxy did not ask for authentication; the configured login was not sent")
		} else {
			rec.skip(StageAuth, "proxy does not require authentication")
		}
		if rec.failed != "" {
			return nil, false
		}

		rec.run(StageConnect, func() (string, error) {
			return fmt.Sprintf("tunnel to %s opened", t.Addr), socks5Connect(conn, host, port)
		})
		return nil, false

	case "socks4":
		rec.skip(StageHandshake, "SOCKS4 has no separate negotiation step")
		if p.Password != "" {
			rec.skip(StageAuth, "SOCKS4 has no password authentication; the configured password was not sent")
		} else {
			rec.skip(StageAuth, "SOCKS4 has no password authentication")
		}
		rec.run(StageConnect, func() (string, error) {
			return fmt.Sprintf("tunnel to %s opened", t.Addr), socks4Connect(conn, host, port, p.Login)
		})
		return nil, false

	case "http-connect":
		rec.skip(StageHandshake, "HTTP proxies negotiate within the CONNECT request")
		var br *bufio.Reader
		var connErr error
		rec.run(StageConnect, func() (string, error) {
			var err error
			br, err = httpConnect(conn, t.Addr, p.Login, p.Password)
			connErr = err
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("tunnel to %s opened", t.Addr), nil
		})
		// Re-attribute an authentication rejection to the auth stage, where an
		// operator will look for it.
		switch {
		case isAuthStage(connErr):
			retagAuth(rec)
		case rec.failed != "":
			// The failure was something else; leave it where it landed.
		case p.Login != "" || p.Password != "":
			// The CONNECT carried the credentials and succeeded, so they were
			// accepted. That is worth showing as a pass rather than a skip.
			rec.pass(StageAuth, "credentials accepted")
		default:
			rec.skip(StageAuth, "proxy did not require authentication")
		}
		return br, false

	case "http-relay":
		rec.skip(StageHandshake, "HTTP relay proxies take a full request per connection")
		if p.Login != "" || p.Password != "" {
			rec.skip(StageAuth, "credentials sent with the relayed request")
		} else {
			rec.skip(StageAuth, "no credentials configured")
		}
		var br *bufio.Reader
		rec.run(StageConnect, func() (string, error) {
			var err error
			br, err = httpRelayRequest(conn, host, port, p.Login, p.Password)
			if err != nil {
				return "", err
			}
			return "request relayed", nil
		})
		return br, true

	default:
		rec.run(StageHandshake, func() (string, error) {
			return "", fmt.Errorf("unknown proxy type %q", p.Type)
		})
		return nil, false
	}
}

// retagAuth moves the most recent failure from the connect stage to the auth
// stage, keeping stage order intact.
func retagAuth(rec *recorder) {
	for i := range rec.stages {
		if rec.stages[i].Name == StageConnect && !rec.stages[i].OK {
			rec.stages[i].Name = StageAuth
			rec.stages[i].Guidance = guidance(StageAuth, fmt.Errorf("%s", rec.stages[i].Error))
			rec.failed = StageAuth
			return
		}
	}
}

// httpRelayRequest sends an absolute-form request, which is how a relay proxy
// expects to receive traffic.
func httpRelayRequest(conn net.Conn, host string, port int, user, pass string) (*bufio.Reader, error) {
	target := host
	if port != 80 {
		target = net.JoinHostPort(host, fmt.Sprint(port))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "HEAD http://%s/ HTTP/1.1\r\n", target)
	fmt.Fprintf(&b, "Host: %s\r\n", target)
	b.WriteString("User-Agent: netcfgd\r\nConnection: close\r\n")
	if user != "" || pass != "" {
		b.WriteString(basicAuthHeader(user, pass))
	}
	b.WriteString("\r\n")

	if _, err := conn.Write([]byte(b.String())); err != nil {
		return nil, fmt.Errorf("send relayed request: %w", err)
	}
	return bufio.NewReader(conn), nil
}

func readRelayResponse(br *bufio.Reader) (string, error) {
	if br == nil {
		return "", fmt.Errorf("no response from relay proxy")
	}
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodHead})
	if err != nil {
		return "", fmt.Errorf("read relayed response: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusProxyAuthRequired {
		return "", fmt.Errorf("proxy requires authentication (407)")
	}
	return resp.Status, nil
}

// ipNetwork maps a dial network to the family LookupIP expects.
func ipNetwork(network string) string {
	switch network {
	case "tcp4":
		return "ip4"
	case "tcp6":
		return "ip6"
	default:
		return "ip"
	}
}

func tlsVersion(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	}
	return fmt.Sprintf("TLS 0x%04x", v)
}
