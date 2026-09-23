package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/normalframework/netcfgd/internal/diag"
	"github.com/normalframework/netcfgd/internal/netmgr"
	"github.com/normalframework/netcfgd/internal/proxyprobe"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
)

func (a *App) apiStatus(w http.ResponseWriter, r *http.Request, s session) {
	st := a.state.State()
	cfg := a.store.Config()

	out := map[string]any{
		"mode":         cfg.Proxy.Mode,
		"frozen":       st.Frozen,
		"freezeReason": st.FreezeReason,
		"lastRunAt":    st.LastRunAt,
		"lastError":    st.LastError,
		"lastProbeOk":  st.LastProbeOK,
		"fallback":     st.FallbackOutcome,
		"passes":       st.Passes,
		"failures":     st.Failures,
		"lastChangeAt": st.LastChangeAt,
	}
	if a.nm != nil {
		out["connectivity"] = a.nm.Connectivity()
	}
	if pc := a.currentPending(); pc != nil {
		out["pending"] = map[string]any{
			"iface":       pc.Iface,
			"secondsLeft": int(time.Until(pc.Deadline).Seconds()),
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) apiInterfaces(w http.ResponseWriter, r *http.Request, s session) {
	if a.nm == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "NetworkManager is not reachable"})
		return
	}
	ifaces, err := a.nm.Interfaces()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ifaces)
}

func (a *App) apiWifiScan(w http.ResponseWriter, r *http.Request, s session) {
	if a.nm == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "NetworkManager is not reachable"})
		return
	}
	networks, err := a.nm.ScanWifi(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, networks)
}

// apiProxyTest runs a stage-by-stage probe against a candidate proxy without
// changing anything on the host. This is the endpoint that makes it safe to
// configure a proxy: an operator can prove the settings work before the
// watchdog applies them and restarts every container.
func (a *App) apiProxyTest(w http.ResponseWriter, r *http.Request, s session) {
	var req struct {
		Type     string `json:"type"`
		IP       string `json:"ip"`
		Port     int    `json:"port"`
		Login    string `json:"login"`
		Password string `json:"password"`
		Target   string `json:"target"`
		Direct   bool   `json:"direct"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not read the request: " + err.Error()})
		return
	}

	target := req.Target
	if target == "" {
		target = a.store.Config().Watchdog.ProbeTarget
	}
	kind := proxyprobe.KindHTTPS
	if _, port, err := splitAddr(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	} else if port != 443 {
		kind = proxyprobe.KindTCP
	}

	var p *proxyprobe.Proxy
	if !req.Direct {
		// An empty password in the request means "use the stored one", so an
		// operator can retest saved settings without retyping the secret.
		password := req.Password
		if password == "" {
			stored := a.store.Config().Proxy
			if stored.IP == req.IP && stored.Port == req.Port {
				password = stored.Password
			}
		}
		p = &proxyprobe.Proxy{
			Type: req.Type, IP: req.IP, Port: req.Port, Login: req.Login, Password: password,
		}
		if p.IP == "" || p.Port == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enter the proxy address and port first"})
			return
		}
	}

	res := proxyprobe.Probe(r.Context(), p,
		proxyprobe.Target{Name: "manual test", Addr: target, Kind: kind}, proxyprobe.DefaultTimeout)

	a.log.Infof("web", s.User, "proxy test run", res.Summary())
	writeJSON(w, http.StatusOK, res)
}

// ipv6Banner warns when IPv6 is quietly bypassing a configured proxy.
func (a *App) ipv6Banner(r *http.Request) *banner {
	if !a.sup.Available() {
		return nil
	}
	host, err := a.sup.GetProxy(r.Context())
	if err != nil || !host.Enabled() {
		return nil
	}
	if v6 := diag.CheckIPv6Bypass(true); v6.Advice != "" {
		return &banner{"warn", v6.Advice}
	}
	return nil
}

func (a *App) apiEndpoints(w http.ResponseWriter, r *http.Request, s session) {
	p, path := a.probePath(r)
	results := diag.CheckEndpoints(r.Context(), p, diag.Endpoints(), proxyprobe.DefaultTimeout)
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "results": results})
}

// probePath decides how the endpoint checks should reach the outside world.
//
// When the host has a proxy configured, the checks go through it explicitly
// rather than relying on redsocks to intercept them. That matters: redsocks is
// a local redirector, so a transparently-proxied TCP check succeeds as soon as
// it reaches redsocks, and a destination the proxy then refuses still looks
// reachable. Going through the proxy directly puts the refusal on the connect
// stage, which is the whole point of the checklist.
func (a *App) probePath(r *http.Request) (*proxyprobe.Proxy, string) {
	if !a.sup.Available() {
		return nil, "direct (no Supervisor, so no host proxy is configured)"
	}
	host, err := a.sup.GetProxy(r.Context())
	if err != nil || !host.Enabled() {
		return nil, "direct (no proxy is configured on the host)"
	}

	p := &proxyprobe.Proxy{
		Type: string(host.Type), IP: host.IP, Port: int(host.Port), Login: host.Login,
	}
	// The Supervisor never returns the password, so take it from what we
	// stored when the same proxy was configured here.
	if stored := a.store.Config().Proxy; stored.IP == host.IP && stored.Port == int(host.Port) {
		p.Password = stored.Password
		if p.Login == "" {
			p.Login = stored.Login
		}
	}
	return p, fmt.Sprintf("through the host proxy (%s %s:%d)", host.Type, host.IP, host.Port)
}

func (a *App) apiTunnel(w http.ResponseWriter, r *http.Request, s session) {
	// Reach the edge the same way cloudflared would. A plain dial is no good
	// here: with a transparent redirector in place it connects to redsocks and
	// reports success even when the proxy goes on to refuse port 7844.
	probe, path := a.probePath(r)
	res := proxyprobe.Probe(r.Context(), probe, proxyprobe.Target{
		Name: "tunnel edge", Addr: "region1.v2.argotunnel.com:7844", Kind: proxyprobe.KindReachable,
	}, proxyprobe.DefaultTimeout)

	st := diag.CheckTunnel(r.Context(), probe != nil, res.OK)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":   path,
		"edge":   res,
		"status": st,
	})
}

// apiPing streams echo replies as they arrive, so a slow or lossy path is
// visible while it is happening rather than only in a summary at the end.
func (a *App) apiPing(w http.ResponseWriter, r *http.Request, s session) {
	host := r.URL.Query().Get("host")
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))

	sse, err := newSSE(w)
	if err != nil {
		a.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer sse.close()

	err = diag.Ping(r.Context(), host, count, func(reply diag.PingReply) {
		sse.send("reply", reply)
	})
	if err != nil {
		sse.send("error", map[string]string{"error": err.Error()})
	}
	sse.send("done", map[string]bool{"done": true})
}

func (a *App) apiTraceroute(w http.ResponseWriter, r *http.Request, s session) {
	host := r.URL.Query().Get("host")
	maxHops, _ := strconv.Atoi(r.URL.Query().Get("maxHops"))

	sse, err := newSSE(w)
	if err != nil {
		a.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer sse.close()

	err = diag.Traceroute(r.Context(), host, maxHops, func(hop diag.Hop) {
		sse.send("hop", hop)
	})
	if err != nil {
		sse.send("error", map[string]string{"error": err.Error()})
	}
	sse.send("done", map[string]bool{"done": true})
}

func (a *App) apiDNS(w http.ResponseWriter, r *http.Request, s session) {
	writeJSON(w, http.StatusOK, diag.ResolveDNS(r.Context(), r.URL.Query().Get("host")))
}

// apiActivityStream pushes new activity entries to the console.
func (a *App) apiActivityStream(w http.ResponseWriter, r *http.Request, s session) {
	sse, err := newSSE(w)
	if err != nil {
		a.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer sse.close()

	entries, release := a.log.Subscribe()
	defer release()

	// A periodic comment keeps intermediaries from closing an idle stream.
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-entries:
			if !ok {
				return
			}
			sse.send("entry", e)
		case <-keepalive.C:
			sse.comment("keepalive")
		}
	}
}

// bundle is the diagnostic snapshot an operator attaches to a support ticket.
type bundle struct {
	GeneratedAt   time.Time                `json:"generatedAt"`
	Version       string                   `json:"version"`
	Interfaces    []netmgr.Interface       `json:"interfaces,omitempty"`
	InterfaceErr  string                   `json:"interfaceError,omitempty"`
	Connectivity  netmgr.Connectivity      `json:"connectivity,omitempty"`
	Gateway       string                   `json:"gateway,omitempty"`
	Resolvers     []string                 `json:"resolvers,omitempty"`
	Device        supervisor.DeviceState   `json:"device"`
	HostProxy     supervisor.Proxy         `json:"hostProxy"`
	Config        store.Config             `json:"config"`
	WatchdogState store.State              `json:"watchdogState"`
	ProbePath     string                   `json:"probePath"`
	Endpoints     []diag.EndpointResult    `json:"endpoints"`
	Tunnel        diag.TunnelStatus        `json:"tunnel"`
	Activity      []activityEntryForBundle `json:"activity"`
}

type activityEntryForBundle struct {
	At      time.Time `json:"at"`
	Level   string    `json:"level"`
	Source  string    `json:"source"`
	Actor   string    `json:"actor,omitempty"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
}

// apiBundle collects everything needed to diagnose a site remotely into one
// downloadable file. Secrets are redacted: this is meant to be emailed.
func (a *App) apiBundle(w http.ResponseWriter, r *http.Request, s session) {
	b := bundle{
		GeneratedAt:   time.Now(),
		Version:       a.opts.Version,
		Gateway:       diag.DefaultGateway(),
		Resolvers:     diag.Resolvers(),
		Config:        a.store.Config(),
		WatchdogState: a.state.State(),
	}

	if a.nm != nil {
		ifaces, err := a.nm.Interfaces()
		if err != nil {
			b.InterfaceErr = err.Error()
		}
		b.Interfaces = ifaces
		b.Connectivity = a.nm.Connectivity()
	}
	if a.sup.Available() {
		b.Device, _ = a.sup.Device(r.Context())
		b.HostProxy, _ = a.sup.GetProxy(r.Context())
	}

	probe, path := a.probePath(r)
	b.ProbePath = path
	b.Endpoints = diag.CheckEndpoints(r.Context(), probe, diag.Endpoints(), proxyprobe.DefaultTimeout)
	tcpOK := false
	for _, e := range b.Endpoints {
		if e.Endpoint.Group == diag.GroupTunnel && e.Result.OK {
			tcpOK = true
		}
	}
	b.Tunnel = diag.CheckTunnel(r.Context(), b.HostProxy.Enabled(), tcpOK)

	for _, e := range a.log.Entries(300) {
		b.Activity = append(b.Activity, activityEntryForBundle{
			At: e.At, Level: string(e.Level), Source: e.Source, Actor: e.Actor,
			Message: e.Message, Detail: e.Detail,
		})
	}

	// Redact everything secret. A bundle is only useful if people are willing
	// to send it, and it is only safe to send if it carries no credentials.
	b.Config.Auth = store.Auth{Username: b.Config.Auth.Username, MustChange: b.Config.Auth.MustChange}
	if b.Config.Proxy.Password != "" {
		b.Config.Proxy.Password = "[redacted]"
	}
	b.HostProxy.Password = ""

	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="netcfgd-bundle-%s.json"`, time.Now().Format("20060102-150405")))
	writeJSON(w, http.StatusOK, b)
}

func splitAddr(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("%q must be in host:port form", addr)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("%q has an invalid port", addr)
	}
	return host, port, nil
}
