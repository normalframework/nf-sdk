// Package supervisor talks to the balena Supervisor API.
//
// The Supervisor is the only route to the host's proxy configuration: it owns
// /mnt/boot/system-proxy/redsocks.conf, which a container cannot write directly
// because balena does not permit bind mounting the boot partition. It is also
// the *only* network configuration the Supervisor exposes — there is no
// per-interface endpoint, which is why NIC configuration goes through
// NetworkManager D-Bus instead (see internal/netmgr).
package supervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ProxyType is a redsocks proxy type. These four are the only values the
// Supervisor accepts.
type ProxyType string

const (
	ProxySOCKS4      ProxyType = "socks4"
	ProxySOCKS5      ProxyType = "socks5"
	ProxyHTTPConnect ProxyType = "http-connect"
	ProxyHTTPRelay   ProxyType = "http-relay"
)

// ValidProxyTypes lists the accepted proxy types in UI order.
var ValidProxyTypes = []ProxyType{ProxyHTTPConnect, ProxySOCKS5, ProxySOCKS4, ProxyHTTPRelay}

func (t ProxyType) Valid() bool {
	for _, v := range ValidProxyTypes {
		if v == t {
			return true
		}
	}
	return false
}

// Port survives the Supervisor's inconsistency between GET and PATCH: PATCH
// takes a JSON number, GET hands back a string.
type Port int

func (p *Port) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*p = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid port %s: %w", b, err)
	}
	*p = Port(n)
	return nil
}

func (p Port) MarshalJSON() ([]byte, error) { return []byte(strconv.Itoa(int(p))), nil }

// Proxy is the host proxy configuration. A zero Proxy means no proxy.
type Proxy struct {
	Type     ProxyType `json:"type,omitempty"`
	IP       string    `json:"ip,omitempty"`
	Port     Port      `json:"port,omitempty"`
	Login    string    `json:"login,omitempty"`
	Password string    `json:"password,omitempty"`
	NoProxy  []string  `json:"noProxy,omitempty"`
	// DNS configures redsocks' dnsu2t plugin, which forces DNS over TCP
	// through the proxy. It is a bool or an "ADDRESS:PORT" string, and cannot
	// be set without a proxy.
	DNS any `json:"dns,omitempty"`
}

// Enabled reports whether this describes an active proxy.
func (p Proxy) Enabled() bool { return p.IP != "" && p.Port != 0 && p.Type != "" }

// Validate checks a proxy configuration before it reaches the Supervisor.
func (p Proxy) Validate() error {
	if !p.Enabled() {
		return nil
	}
	if !p.Type.Valid() {
		return fmt.Errorf("proxy type %q must be one of socks4, socks5, http-connect, http-relay", p.Type)
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("proxy port %d is out of range", p.Port)
	}
	if p.DNS != nil && !p.Enabled() {
		return fmt.Errorf("dnsu2t cannot be configured without a proxy")
	}
	return nil
}

// Addr is the proxy's host:port.
func (p Proxy) Addr() string { return fmt.Sprintf("%s:%d", p.IP, p.Port) }

// Normalized returns the canonical form of a proxy configuration: the form
// that will actually exist on the host once it is applied.
//
// The Supervisor is told to exclude the proxy's own address from noProxy, so a
// configuration read back from the host always contains it. Normalizing in one
// place — rather than only on the way out — is what lets the watchdog compare
// what it wants against what the host has and correctly conclude "no change
// needed". Without it the comparison never matches, and the watchdog re-applies
// the proxy on every pass, restarting every container on the device each time.
func (p Proxy) Normalized() Proxy {
	if !p.Enabled() {
		return Proxy{}
	}
	seen := map[string]bool{}
	var noProxy []string
	for _, e := range append(append([]string(nil), p.NoProxy...), p.IP) {
		e = strings.TrimSpace(e)
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		noProxy = append(noProxy, e)
	}
	sort.Strings(noProxy)
	p.NoProxy = noProxy
	return p
}

// SameAs compares two proxy configurations using only the fields the Supervisor
// echoes back from GET. The Supervisor never returns the password, so password
// changes cannot be detected by comparison — callers that need that must track
// what they applied themselves (see internal/watchdog).
//
// This comparison is what keeps the watchdog from issuing a redundant PATCH,
// and a redundant PATCH restarts balenaEngine and therefore every container on
// the device. It is deliberately conservative.
func (p Proxy) SameAs(other Proxy) bool {
	if p.Enabled() != other.Enabled() {
		return false
	}
	if !p.Enabled() {
		return true
	}
	p, other = p.Normalized(), other.Normalized()
	if p.Type != other.Type || p.IP != other.IP || p.Port != other.Port || p.Login != other.Login {
		return false
	}
	if !sameStrings(p.NoProxy, other.NoProxy) {
		return false
	}
	return normalizeDNS(p.DNS) == normalizeDNS(other.DNS)
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// normalizeDNS collapses the bool/string union into a comparable string.
func normalizeDNS(v any) string {
	switch d := v.(type) {
	case nil:
		return ""
	case bool:
		if d {
			return "8.8.8.8:53" // the Supervisor's documented default
		}
		return ""
	case string:
		return d
	default:
		return fmt.Sprint(d)
	}
}

// HostConfig is the body of /v1/device/host-config.
type HostConfig struct {
	Network Network `json:"network"`
}

type Network struct {
	Proxy    *Proxy `json:"proxy,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Force    bool   `json:"force,omitempty"`
}

// DeviceState is the subset of /v1/device that is useful here.
type DeviceState struct {
	Status           string `json:"status"`
	IPAddress        string `json:"ip_address"`
	MACAddress       string `json:"mac_address"`
	OSVersion        string `json:"os_version"`
	SupervisorVer    string `json:"supervisor_version"`
	Commit           string `json:"commit"`
	UpdatePending    bool   `json:"update_pending"`
	UpdateDownloaded bool   `json:"update_downloaded"`
	UpdateFailed     bool   `json:"update_failed"`
}

// API is the surface the watchdog depends on, so it can be faked in tests.
type API interface {
	GetProxy(ctx context.Context) (Proxy, error)
	SetProxy(ctx context.Context, p Proxy, force bool) error
	ClearProxy(ctx context.Context, force bool) error
	Reboot(ctx context.Context, force bool) error
	Available() bool
}

// Client is a balena Supervisor API client.
type Client struct {
	base   string
	apiKey string
	http   *http.Client

	// DryRun logs mutating calls instead of making them. Set it when working
	// against a real NetworkManager on a development machine, so the proxy
	// logic can be exercised without a balena device.
	DryRun bool
}

// New builds a client from BALENA_SUPERVISOR_ADDRESS and
// BALENA_SUPERVISOR_API_KEY, which the io.balena.features.supervisor-api label
// injects into the container.
func New() *Client {
	return &Client{
		base:   strings.TrimSuffix(os.Getenv("BALENA_SUPERVISOR_ADDRESS"), "/"),
		apiKey: os.Getenv("BALENA_SUPERVISOR_API_KEY"),
		// The Supervisor can be slow while it holds update locks, and a PATCH
		// restarts balenaEngine underneath us, so be patient rather than
		// retrying an operation that may already have taken effect.
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

// Available reports whether the Supervisor API is configured. It is not when
// running outside balena, which is the normal case during development.
func (c *Client) Available() bool { return c.base != "" && c.apiKey != "" }

func (c *Client) url(path string) string {
	return fmt.Sprintf("%s%s?apikey=%s", c.base, path, c.apiKey)
}

// transientAttempts is how many times an idempotent request is tried before
// the failure is reported.
//
// The Supervisor is routinely unavailable for a few seconds: it restarts on
// every boot, on every OS update, and after every host-config change we make.
// On balenaOS the address we talk to is a host-side proxy that forwards to the
// Supervisor container, so a restart shows up as a 502 "Target connection
// failed" or a reset connection rather than a clean refusal. None of that is
// worth showing an operator as an error.
const transientAttempts = 3

// transient reports whether an error or status is worth retrying.
func transient(err error, status int) bool {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) || errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// http.Client wraps transport failures in *url.Error; the string check
	// catches resets that do not surface as a typed syscall error.
	msg := err.Error()
	return strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "connection refused")
}

// do issues a request, retrying idempotent ones through a Supervisor restart.
//
// Only GET is retried. A PATCH that appears to fail may already have taken
// effect, and re-sending it would restart every container on the device a
// second time; the watchdog's pending-intent record handles that case instead.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	if method != http.MethodGet {
		return c.doOnce(ctx, method, path, body, out)
	}

	var err error
	for attempt := 1; attempt <= transientAttempts; attempt++ {
		err = c.doOnce(ctx, method, path, body, out)
		if err == nil {
			return nil
		}
		var te transientError
		if !errors.As(err, &te) || attempt == transientAttempts {
			return err
		}
		// Short, increasing pause. The Supervisor is usually back within a
		// couple of seconds of a restart.
		select {
		case <-ctx.Done():
			return err
		case <-time.After(time.Duration(attempt) * 750 * time.Millisecond):
		}
	}
	return err
}

// transientError marks a failure that a retry might clear.
type transientError struct{ err error }

func (e transientError) Error() string { return e.err.Error() }
func (e transientError) Unwrap() error { return e.err }

func (c *Client) doOnce(ctx context.Context, method, path string, body any, out any) error {
	if !c.Available() {
		return fmt.Errorf("supervisor API unavailable: BALENA_SUPERVISOR_ADDRESS and BALENA_SUPERVISOR_API_KEY are not set (is the io.balena.features.supervisor-api label present?)")
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url(path), reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		wrapped := fmt.Errorf("%s %s: %w", method, path, err)
		if transient(err, 0) {
			return transientError{wrapped}
		}
		return wrapped
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 423 is the Supervisor refusing because an update lock is held. Say
		// so plainly; it is a normal condition, not a failure of this app.
		if resp.StatusCode == http.StatusLocked {
			return fmt.Errorf("supervisor refused the change: an update lock is held (retry, or enable force)")
		}
		wrapped := fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(data)))
		if transient(nil, resp.StatusCode) {
			return transientError{wrapped}
		}
		return wrapped
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s response %q: %w", path, string(data), err)
		}
	}
	return nil
}

// GetProxy reads the currently configured host proxy.
func (c *Client) GetProxy(ctx context.Context) (Proxy, error) {
	var hc HostConfig
	if err := c.do(ctx, http.MethodGet, "/v1/device/host-config", nil, &hc); err != nil {
		return Proxy{}, err
	}
	if hc.Network.Proxy == nil {
		return Proxy{}, nil
	}
	return *hc.Network.Proxy, nil
}

// SetProxy applies a proxy configuration.
//
// This restarts balenaEngine on balenaOS 2.82.6 and newer, which restarts every
// container on the device including this one. Callers must confirm the change
// is actually needed before calling.
func (c *Client) SetProxy(ctx context.Context, p Proxy, force bool) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if c.DryRun {
		log.Printf("supervisor: DRY RUN, would set proxy %s %s (force=%v)", p.Type, p.Addr(), force)
		return nil
	}
	// Never proxy the connection to the proxy itself. Without this, a proxy on
	// a public address would have our probe traffic redirected back into
	// redsocks and loop.
	p = p.Normalized()
	body := HostConfig{Network: Network{Proxy: &p, Force: force}}
	return c.do(ctx, http.MethodPatch, "/v1/device/host-config", body, nil)
}

// ClearProxy removes the host proxy configuration. An empty proxy object is the
// Supervisor's documented way to reset it.
func (c *Client) ClearProxy(ctx context.Context, force bool) error {
	if c.DryRun {
		log.Printf("supervisor: DRY RUN, would clear the proxy (force=%v)", force)
		return nil
	}
	body := HostConfig{Network: Network{Proxy: &Proxy{}, Force: force}}
	return c.do(ctx, http.MethodPatch, "/v1/device/host-config", body, nil)
}

// SetHostname changes the device hostname.
func (c *Client) SetHostname(ctx context.Context, hostname string, force bool) error {
	if c.DryRun {
		log.Printf("supervisor: DRY RUN, would set hostname to %q", hostname)
		return nil
	}
	body := HostConfig{Network: Network{Hostname: hostname, Force: force}}
	return c.do(ctx, http.MethodPatch, "/v1/device/host-config", body, nil)
}

// GetHostname reads the current device hostname.
func (c *Client) GetHostname(ctx context.Context) (string, error) {
	var hc HostConfig
	if err := c.do(ctx, http.MethodGet, "/v1/device/host-config", nil, &hc); err != nil {
		return "", err
	}
	return hc.Network.Hostname, nil
}

// Device reads the Supervisor's view of the device.
func (c *Client) Device(ctx context.Context) (DeviceState, error) {
	var d DeviceState
	err := c.do(ctx, http.MethodGet, "/v1/device", nil, &d)
	return d, err
}

// Version returns the Supervisor version.
func (c *Client) Version(ctx context.Context) (string, error) {
	var v struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := c.do(ctx, http.MethodGet, "/v2/version", nil, &v); err != nil {
		return "", err
	}
	return v.Version, nil
}

// Reboot reboots the device. This is the only way to force connections that
// were established before a proxy change to go through the proxy: the
// Supervisor docs note that existing connections are not closed when the proxy
// changes.
func (c *Client) Reboot(ctx context.Context, force bool) error {
	if c.DryRun {
		log.Printf("supervisor: DRY RUN, would reboot the device (force=%v)", force)
		return nil
	}
	body := map[string]bool{"force": force}
	return c.do(ctx, http.MethodPost, "/v1/reboot", body, nil)
}
