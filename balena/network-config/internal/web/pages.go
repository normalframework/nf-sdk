package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/normalframework/netcfgd/internal/diag"
	"github.com/normalframework/netcfgd/internal/netmgr"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
)

func (a *App) getLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.readSession(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	a.render(w, "login.html", pageData{
		Title:   "Sign in",
		Version: a.opts.Version,
		Flash:   a.takeFlash(w, r),
		Data:    map[string]any{"Next": r.URL.Query().Get("next")},
	})
}

func (a *App) postLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.fail(w, r, http.StatusBadRequest, "Could not read the submitted form.")
		return
	}

	ip := clientIP(r)
	if ok, wait := a.limiter.allow(ip); !ok {
		a.log.Warnf("web", ip, "login blocked by rate limit", fmt.Sprintf("retry in %s", wait.Round(time.Second)))
		a.redirect(w, r, "/login", "error",
			fmt.Sprintf("Too many failed sign-in attempts. Try again in %s.", wait.Round(time.Second)))
		return
	}

	user := r.FormValue("username")
	if !a.store.Verify(user, r.FormValue("password")) {
		a.limiter.record(ip)
		a.log.Warnf("web", ip, "failed sign-in attempt", "username "+user)
		a.redirect(w, r, "/login", "error", "Incorrect username or password.")
		return
	}

	a.limiter.reset(ip)
	// Drop any leftover failure message, so a successful sign-in never lands
	// on a page still saying the password was wrong.
	a.takeFlash(w, r)
	if err := a.issueSession(w, user); err != nil {
		a.fail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	a.log.Infof("web", user, "signed in", "from "+ip)

	next := r.FormValue("next")
	if next == "" || next[0] != '/' {
		next = "/"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *App) postLogout(w http.ResponseWriter, r *http.Request, s session) {
	a.clearSession(w)
	a.log.Infof("web", s.User, "signed out", "")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// overviewData is the at-a-glance state of the device.
type overviewData struct {
	Interfaces   []netmgr.Interface
	Connectivity netmgr.Connectivity
	Gateway      string
	Resolvers    []string
	Proxy        supervisor.Proxy
	ProxyErr     string
	Mode         store.Mode
	State        store.State
	Device       supervisor.DeviceState
	DeviceErr    string
	Uptime       string
}

func (a *App) getOverview(w http.ResponseWriter, r *http.Request, s session) {
	ctx := r.Context()
	d := overviewData{
		Mode:      a.store.Config().Proxy.Mode,
		State:     a.state.State(),
		Gateway:   diag.DefaultGateway(),
		Resolvers: diag.Resolvers(),
	}

	if a.nm != nil {
		if ifaces, err := a.nm.Interfaces(); err == nil {
			d.Interfaces = ifaces
		}
		d.Connectivity = a.nm.Connectivity()
	}
	if a.sup.Available() {
		p, err := a.sup.GetProxy(ctx)
		if err != nil {
			d.ProxyErr = err.Error()
		} else {
			d.Proxy = p
		}
		dev, err := a.sup.Device(ctx)
		if err != nil {
			d.DeviceErr = err.Error()
		} else {
			d.Device = dev
		}
	}
	if up, err := store.Uptime(); err == nil {
		d.Uptime = up.Round(time.Second).String()
	}

	a.render(w, "overview.html", a.page(w, r, s, "overview", "Overview", d))
}

type interfacesData struct {
	Interfaces []netmgr.Interface
	Pending    *pendingChange
	NMVersion  string
}

func (a *App) getInterfaces(w http.ResponseWriter, r *http.Request, s session) {
	d := interfacesData{Pending: a.currentPending()}
	if a.nm != nil {
		ifaces, err := a.nm.Interfaces()
		if err != nil {
			a.setFlash(w, "error", "Could not read interfaces: "+err.Error())
		}
		d.Interfaces = ifaces
		d.NMVersion, _ = a.nm.Version()
	}
	a.render(w, "interfaces.html", a.page(w, r, s, "interfaces", "Interfaces", d))
}

type proxyData struct {
	Config    store.ProxyConfig
	Watchdog  store.WatchdogConfig
	State     store.State
	Host      supervisor.Proxy
	HostErr   string
	Types     []supervisor.ProxyType
	Modes     []store.Mode
	SupVer    string
	SupVerErr string
}

func (a *App) getProxy(w http.ResponseWriter, r *http.Request, s session) {
	cfg := a.store.Config()
	d := proxyData{
		Config:   cfg.Proxy,
		Watchdog: cfg.Watchdog,
		State:    a.state.State(),
		Types:    supervisor.ValidProxyTypes,
		Modes:    []store.Mode{store.ModeOff, store.ModeStatic, store.ModeFallback, store.ModeDynamic},
	}
	// The password is never sent back to the browser; the form shows whether
	// one is set instead.
	d.Config.Password = ""

	if a.sup.Available() {
		p, err := a.sup.GetProxy(r.Context())
		if err != nil {
			d.HostErr = err.Error()
		} else {
			d.Host = p
		}
		if v, err := a.sup.Version(r.Context()); err == nil {
			d.SupVer = v
		} else {
			d.SupVerErr = err.Error()
		}
	}
	a.render(w, "proxy.html", a.page(w, r, s, "proxy", "Proxy", d))
}

type diagnosticsData struct {
	Endpoints []diag.Endpoint
	Gateway   string
	Resolvers []string
	ProxySet  bool
}

func (a *App) getDiagnostics(w http.ResponseWriter, r *http.Request, s session) {
	d := diagnosticsData{
		Endpoints: diag.Endpoints(),
		Gateway:   diag.DefaultGateway(),
		Resolvers: diag.Resolvers(),
	}
	if a.sup.Available() {
		if p, err := a.sup.GetProxy(r.Context()); err == nil {
			d.ProxySet = p.Enabled()
		}
	}
	a.render(w, "diagnostics.html", a.page(w, r, s, "diagnostics", "Diagnostics", d))
}

func (a *App) getActivity(w http.ResponseWriter, r *http.Request, s session) {
	entries := a.log.Entries(300)
	a.render(w, "activity.html", a.page(w, r, s, "activity", "Activity", entries))
}

type settingsData struct {
	Auth     store.Auth
	Hostname string
	HostErr  string
	Config   store.Config
}

func (a *App) getSettings(w http.ResponseWriter, r *http.Request, s session) {
	d := settingsData{Auth: a.store.Auth(), Config: a.store.Config()}
	if a.sup.Available() {
		h, err := a.sup.GetHostname(r.Context())
		if err != nil {
			d.HostErr = err.Error()
		} else {
			d.Hostname = h
		}
	}
	a.render(w, "settings.html", a.page(w, r, s, "settings", "Settings", d))
}

func (a *App) postPassword(w http.ResponseWriter, r *http.Request, s session) {
	if err := r.ParseForm(); err != nil {
		a.fail(w, r, http.StatusBadRequest, "Could not read the submitted form.")
		return
	}
	// The password form is reachable while MustChange is set, which is before
	// the mutate wrapper would run, so CSRF is checked here explicitly.
	if !a.checkCSRF(r, s) {
		a.fail(w, r, http.StatusForbidden, "That form has expired. Reload the page and try again.")
		return
	}
	if a.opts.ReadOnly {
		a.fail(w, r, http.StatusForbidden, "This console is running in read-only mode.")
		return
	}

	current := r.FormValue("current")
	username := r.FormValue("username")
	if username == "" {
		username = a.store.Auth().Username
	}
	if !a.store.Verify(a.store.Auth().Username, current) {
		a.redirect(w, r, "/settings", "error", "The current password is not correct.")
		return
	}
	if r.FormValue("password") != r.FormValue("confirm") {
		a.redirect(w, r, "/settings", "error", "The two new passwords do not match.")
		return
	}
	if err := a.store.SetPassword(username, r.FormValue("password")); err != nil {
		a.redirect(w, r, "/settings", "error", err.Error())
		return
	}

	a.log.Changef("web", s.User, "console password changed", "all sessions have been signed out")
	// Changing the password rotates the session key, so this session is gone.
	a.clearSession(w)
	a.redirect(w, r, "/login", "ok", "Password changed. Sign in again with the new password.")
}
