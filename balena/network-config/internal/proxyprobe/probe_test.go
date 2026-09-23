package proxyprobe

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// socksServer is a minimal SOCKS5 server for tests. authRequired makes it
// demand username/password; refuse makes it reject the CONNECT.
type socksServer struct {
	ln           net.Listener
	authRequired bool
	user, pass   string
	refuse       bool
}

func newSocksServer(t *testing.T, s *socksServer) *socksServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s.ln = ln
	t.Cleanup(func() { ln.Close() })
	go s.serve()
	return s
}

func (s *socksServer) addr() (string, int) {
	host, portStr, _ := net.SplitHostPort(s.ln.Addr().String())
	var port int
	binary.Size(port)
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	return host, port
}

func (s *socksServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *socksServer) handle(conn net.Conn) {
	defer conn.Close()

	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		return
	}
	methods := make([]byte, head[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}

	if s.authRequired {
		conn.Write([]byte{0x05, 0x02})
		var vh [2]byte
		if _, err := io.ReadFull(conn, vh[:]); err != nil {
			return
		}
		user := make([]byte, vh[1])
		io.ReadFull(conn, user)
		var pl [1]byte
		io.ReadFull(conn, pl[:])
		pass := make([]byte, pl[0])
		io.ReadFull(conn, pass)

		if string(user) != s.user || string(pass) != s.pass {
			conn.Write([]byte{0x01, 0x01}) // failure
			return
		}
		conn.Write([]byte{0x01, 0x00})
	} else {
		conn.Write([]byte{0x05, 0x00})
	}

	// CONNECT request.
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	var host string
	switch req[3] {
	case 0x01:
		b := make([]byte, 4)
		io.ReadFull(conn, b)
		host = net.IP(b).String()
	case 0x03:
		var l [1]byte
		io.ReadFull(conn, l[:])
		b := make([]byte, l[0])
		io.ReadFull(conn, b)
		host = string(b)
	}
	var portB [2]byte
	io.ReadFull(conn, portB[:])
	port := binary.BigEndian.Uint16(portB[:])

	if s.refuse {
		conn.Write([]byte{0x05, 0x02, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // not allowed by ruleset
		return
	}

	upstream, err := net.Dial("tcp", net.JoinHostPort(host, itoa(int(port))))
	if err != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // host unreachable
		return
	}
	defer upstream.Close()
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	go io.Copy(upstream, conn)
	io.Copy(conn, upstream)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// stage returns the recorded stage by name.
func stage(t *testing.T, r Result, name string) Stage {
	t.Helper()
	for _, s := range r.Stages {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("probe did not record a %q stage; stages were %v", name, stageNames(r))
	return Stage{}
}

func stageNames(r Result) []string {
	var out []string
	for _, s := range r.Stages {
		out = append(out, s.Name)
	}
	return out
}

// echoServer is a plain TCP listener used as a CONNECT target.
func echoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()
	return ln.Addr().String()
}

func TestSOCKS5NoAuthReachesTarget(t *testing.T) {
	srv := newSocksServer(t, &socksServer{})
	host, port := srv.addr()
	target := echoServer(t)

	res := Probe(context.Background(), &Proxy{Type: "socks5", IP: host, Port: port},
		Target{Name: "echo", Addr: target, Kind: KindTCP}, 5*time.Second)

	if !res.OK {
		t.Fatalf("probe failed: %s", res.Summary())
	}
	if s := stage(t, res, StageHandshake); !s.OK {
		t.Fatalf("handshake stage failed: %s", s.Error)
	}
	if s := stage(t, res, StageConnect); !s.OK {
		t.Fatalf("connect stage failed: %s", s.Error)
	}
}

func TestSOCKS5CorrectCredentialsAuthenticate(t *testing.T) {
	srv := newSocksServer(t, &socksServer{authRequired: true, user: "u", pass: "p"})
	host, port := srv.addr()
	target := echoServer(t)

	res := Probe(context.Background(),
		&Proxy{Type: "socks5", IP: host, Port: port, Login: "u", Password: "p"},
		Target{Addr: target, Kind: KindTCP}, 5*time.Second)

	if !res.OK {
		t.Fatalf("probe failed: %s", res.Summary())
	}
	if s := stage(t, res, StageAuth); !s.OK {
		t.Fatalf("auth stage failed: %s", s.Error)
	}
}

// A wrong password must be reported against the auth stage, not buried in a
// generic connection failure — that distinction is the point of the prober.
func TestSOCKS5WrongCredentialsFailAtAuthStage(t *testing.T) {
	srv := newSocksServer(t, &socksServer{authRequired: true, user: "u", pass: "p"})
	host, port := srv.addr()

	res := Probe(context.Background(),
		&Proxy{Type: "socks5", IP: host, Port: port, Login: "u", Password: "wrong"},
		Target{Addr: "example.com:443", Kind: KindTCP}, 5*time.Second)

	if res.OK {
		t.Fatal("probe succeeded with the wrong password")
	}
	if res.Failed != StageAuth {
		t.Fatalf("failed at stage %q, want %q (%s)", res.Failed, StageAuth, res.Summary())
	}
	if s := stage(t, res, StageAuth); !strings.Contains(s.Guidance, "username and password") {
		t.Fatalf("auth guidance did not mention credentials: %q", s.Guidance)
	}
	// Everything after the failure must be recorded as skipped, not omitted.
	if s := stage(t, res, StageConnect); !s.Skipped {
		t.Fatalf("connect stage should be skipped after an auth failure, got %+v", s)
	}
}

// A proxy that connects and authenticates but refuses the destination is the
// "ask IT to allow this host" case, and must be attributed to connect.
func TestSOCKS5RefusedDestinationFailsAtConnectStage(t *testing.T) {
	srv := newSocksServer(t, &socksServer{refuse: true})
	host, port := srv.addr()

	res := Probe(context.Background(), &Proxy{Type: "socks5", IP: host, Port: port},
		Target{Addr: "region1.v2.argotunnel.com:7844", Kind: KindTCP}, 5*time.Second)

	if res.OK {
		t.Fatal("probe succeeded against a refusing proxy")
	}
	if res.Failed != StageConnect {
		t.Fatalf("failed at stage %q, want %q (%s)", res.Failed, StageConnect, res.Summary())
	}
	s := stage(t, res, StageConnect)
	if !strings.Contains(s.Error, "not allowed") {
		t.Fatalf("connect error did not describe the refusal: %q", s.Error)
	}
	if !strings.Contains(s.Guidance, "Blocked by policy") {
		t.Fatalf("connect guidance was not actionable: %q", s.Guidance)
	}
}

func TestUnreachableProxyFailsAtTCPStage(t *testing.T) {
	// Port 1 on loopback: nothing listens, and the connection is refused
	// rather than dropped, so this is fast and deterministic.
	res := Probe(context.Background(), &Proxy{Type: "socks5", IP: "127.0.0.1", Port: 1},
		Target{Addr: "example.com:443", Kind: KindTCP}, 3*time.Second)

	if res.OK {
		t.Fatal("probe succeeded against a closed port")
	}
	if res.Failed != StageTCP {
		t.Fatalf("failed at stage %q, want %q", res.Failed, StageTCP)
	}
	if s := stage(t, res, StageTCP); !strings.Contains(s.Guidance, "Nothing is listening") {
		t.Fatalf("tcp guidance was not actionable: %q", s.Guidance)
	}
}

// http-connect proxies signal authentication with 407, which must land on the
// auth stage even though it arrives in the CONNECT response.
func TestHTTPConnect407FailsAtAuthStage(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		conn.Read(buf)
		conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n" +
			"Proxy-Authenticate: Basic realm=\"corp\"\r\nContent-Length: 0\r\n\r\n"))
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	res := Probe(context.Background(), &Proxy{Type: "http-connect", IP: host, Port: port},
		Target{Addr: "example.com:443", Kind: KindTCP}, 5*time.Second)

	if res.OK {
		t.Fatal("probe succeeded against a proxy demanding authentication")
	}
	if res.Failed != StageAuth {
		t.Fatalf("failed at stage %q, want %q (%s)", res.Failed, StageAuth, res.Summary())
	}
	s := stage(t, res, StageAuth)
	if !strings.Contains(s.Error, "Basic") {
		t.Fatalf("auth error did not name the scheme the proxy asked for: %q", s.Error)
	}
	// A 407 with no credentials configured must not tell the operator to check
	// a password they never set.
	if !strings.Contains(s.Error, "none are configured") {
		t.Fatalf("407 with no credentials should say none are configured, got %q", s.Error)
	}
	if !strings.Contains(s.Guidance, "none are configured") {
		t.Fatalf("guidance should point at the missing credentials, got %q", s.Guidance)
	}
}

// The same 407 with credentials supplied means they were rejected, which is a
// different fix for the operator.
func TestHTTPConnect407WithCredentialsReportsRejection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		conn.Read(buf)
		conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n" +
			"Proxy-Authenticate: Basic realm=\"corp\"\r\nContent-Length: 0\r\n\r\n"))
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	res := Probe(context.Background(),
		&Proxy{Type: "http-connect", IP: host, Port: port, Login: "u", Password: "p"},
		Target{Addr: "example.com:443", Kind: KindTCP}, 5*time.Second)

	if res.Failed != StageAuth {
		t.Fatalf("failed at stage %q, want %q", res.Failed, StageAuth)
	}
	s := stage(t, res, StageAuth)
	if !strings.Contains(s.Error, "rejected the credentials") {
		t.Fatalf("407 with credentials should report rejection, got %q", s.Error)
	}
	if !strings.Contains(s.Guidance, "Check the username and password") {
		t.Fatalf("guidance should point at the credentials, got %q", s.Guidance)
	}
}

// A re-encrypting proxy must be named, not merely reported as a certificate
// error, and the CA it uses has to come back in a form that can be trusted —
// that is the difference between a dead end and an actionable finding.
func TestInterceptedTLSIdentifiesTheCA(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")

	res := Probe(context.Background(), nil, Target{Addr: addr, Kind: KindTLS}, 5*time.Second)

	if res.OK {
		t.Fatal("probe trusted a certificate from an unknown authority")
	}
	if res.Failed != StageTLS {
		t.Fatalf("failed at stage %q, want %q (%s)", res.Failed, StageTLS, res.Summary())
	}
	if s := stage(t, res, StageTLS); !strings.Contains(s.Guidance, "balenaRootCA") {
		t.Fatalf("guidance should say how to trust the CA, got %q", s.Guidance)
	}

	i := res.Interception
	if i == nil {
		t.Fatal("no interception details were captured")
	}
	if i.LeafIssuer == "" {
		t.Fatalf("interception does not name the interceptor: %+v", i)
	}
	if len(i.Chain) == 0 || i.Chain[0].Fingerprint == "" {
		t.Fatalf("the presented chain was not recorded: %+v", i)
	}
	if i.Trusted {
		t.Fatal("an unknown CA must not be reported as trusted")
	}

	// httptest presents a self-signed CA, so trust material is available here
	// and must round-trip exactly, since it is pasted into config.json.
	if !i.CAPresented || i.RootCA == nil {
		t.Fatalf("a self-signed CA was presented but not offered: %+v", i)
	}
	decoded, err := base64.StdEncoding.DecodeString(i.RootCA.Base64)
	if err != nil {
		t.Fatalf("Base64 does not decode: %v", err)
	}
	if string(decoded) != i.RootCA.PEM {
		t.Fatal("Base64 does not round-trip to PEM")
	}
}

// The common case: the interceptor sends a leaf signed by a CA it does not
// send. Nothing may be offered as trust material, because the only thing on
// offer would be a per-host leaf that is useless in a trust store.
func TestInterceptionWithoutTheCAOffersNothingToTrust(t *testing.T) {
	leaf, caPool := issueLeafFromPrivateCA(t, "127.0.0.1")

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{leaf}})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { c.(*tls.Conn).Handshake(); c.Close() }()
		}
	}()
	_ = caPool

	res := Probe(context.Background(), nil,
		Target{Addr: ln.Addr().String(), Kind: KindTLS}, 5*time.Second)

	if res.OK {
		t.Fatal("probe trusted a privately-issued certificate")
	}
	i := res.Interception
	if i == nil {
		t.Fatal("no interception details were captured")
	}
	if i.CAPresented || i.RootCA != nil {
		t.Fatalf("offered trust material that was never presented: %+v", i.RootCA)
	}
	if i.LeafIssuer == "" || !strings.Contains(i.LeafIssuer, "Test Corp") {
		t.Fatalf("did not name the interceptor from the leaf issuer: %q", i.LeafIssuer)
	}
	for _, c := range i.Chain {
		if c.PEM != "" && !(c.IsCA && c.SelfSigned) {
			t.Fatalf("offered PEM for a certificate that is not a self-signed CA: %+v", c)
		}
	}
}

// Once the intercepting CA is trusted, probes must pass and say so, rather than
// silently looking like ordinary public TLS.
func TestTrustedInterceptionPassesAndIsReported(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "https://")

	// Trust the test server's own CA, standing in for a corporate root.
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE", Bytes: srv.Certificate().Raw,
	})
	if err := LoadTrustedCAs(caPEM, "test bundle"); err != nil {
		t.Fatalf("LoadTrustedCAs: %v", err)
	}
	t.Cleanup(func() {
		trustMu.Lock()
		extraRoots, trustNote = nil, ""
		trustMu.Unlock()
	})

	res := Probe(context.Background(), nil, Target{Addr: addr, Kind: KindTLS}, 5*time.Second)
	if !res.OK {
		t.Fatalf("probe failed with the CA trusted: %s", res.Summary())
	}
	if res.Interception == nil || !res.Interception.Trusted {
		t.Fatalf("a privately-rooted chain should be reported as trusted interception, got %+v", res.Interception)
	}
}

// The direct path still records every stage, so the console renders one shape.
func TestDirectProbeSkipsProxyStages(t *testing.T) {
	target := echoServer(t)
	res := Probe(context.Background(), nil, Target{Addr: target, Kind: KindTCP}, 3*time.Second)

	if !res.OK {
		t.Fatalf("direct probe failed: %s", res.Summary())
	}
	for _, name := range []string{StageHandshake, StageAuth, StageConnect} {
		if s := stage(t, res, name); !s.Skipped {
			t.Fatalf("stage %q should be skipped on a direct probe, got %+v", name, s)
		}
	}
	if res.Via != "direct" {
		t.Fatalf("got Via %q, want direct", res.Via)
	}
}
