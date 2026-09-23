// Package web serves netcfgd's local console.
//
// The console is deliberately server-rendered HTML with a small amount of
// vanilla JavaScript for the live panes. There is no bundler, no npm, and no
// vendored framework: the whole UI ships inside the Go binary through go:embed,
// which keeps the container small and means there is nothing to build.
package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/normalframework/netcfgd/internal/activity"
	"github.com/normalframework/netcfgd/internal/netmgr"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
	"github.com/normalframework/netcfgd/internal/watchdog"
)

//go:embed templates/*.html static/*
var assets embed.FS

// Options configures the server.
type Options struct {
	// ReadOnly disables every mutating endpoint. Useful once a site is
	// commissioned and the console should only be a diagnostic view.
	ReadOnly bool
	// Version is shown in the footer.
	Version string
}

// App holds the console's dependencies.
type App struct {
	store   *store.Store
	state   *store.StateStore
	nm      *netmgr.Manager
	nmErr   error
	sup     *supervisor.Client
	dog     *watchdog.Watchdog
	log     *activity.Log
	opts    Options
	tmpl    *template.Template
	limiter *loginLimiter

	mu      sync.Mutex
	pending *pendingChange
}

// pendingChange is an interface change awaiting confirmation.
//
// Reconfiguring the interface an operator is browsing through will disconnect
// them, and on a device in a plant room that can mean a site visit. So a change
// is applied, and reverted automatically unless the browser comes back and
// confirms it — the same guard a decent router offers.
type pendingChange struct {
	Iface    string
	Snapshot netmgr.Snapshot
	Deadline time.Time
	Token    string
	cancel   context.CancelFunc
}

// New builds the console.
func New(st *store.Store, state *store.StateStore, nm *netmgr.Manager, nmErr error,
	sup *supervisor.Client, dog *watchdog.Watchdog, log *activity.Log, opts Options) (*App, error) {

	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &App{
		store: st, state: state, nm: nm, nmErr: nmErr, sup: sup, dog: dog,
		log: log, opts: opts, tmpl: tmpl,
		// Five attempts per quarter hour: enough for a typo, useless for a
		// dictionary.
		limiter: newLoginLimiter(5, 15*time.Minute),
	}, nil
}

// Handler returns the console's HTTP handler.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	static, _ := fsSub(assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheControl(http.FileServer(http.FS(static)))))

	// Unauthenticated.
	mux.HandleFunc("GET /login", a.getLogin)
	mux.HandleFunc("POST /login", a.postLogin)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})

	// Authenticated pages.
	mux.Handle("GET /{$}", a.auth(a.getOverview))
	mux.Handle("GET /interfaces", a.auth(a.getInterfaces))
	mux.Handle("GET /proxy", a.auth(a.getProxy))
	mux.Handle("GET /diagnostics", a.auth(a.getDiagnostics))
	mux.Handle("GET /activity", a.auth(a.getActivity))
	mux.Handle("GET /settings", a.auth(a.getSettings))
	mux.Handle("POST /logout", a.auth(a.postLogout))

	// Mutations.
	mux.Handle("POST /password", a.auth(a.postPassword))
	mux.Handle("POST /interfaces/{iface}/ipv4", a.mutate(a.postIPv4))
	mux.Handle("POST /interfaces/{iface}/state", a.mutate(a.postIfaceState))
	mux.Handle("POST /interfaces/{iface}/wifi", a.mutate(a.postWifi))
	mux.Handle("POST /interfaces/priority", a.mutate(a.postPriority))
	mux.Handle("POST /interfaces/confirm", a.mutate(a.postConfirm))
	mux.Handle("POST /interfaces/revert", a.mutate(a.postRevert))
	mux.Handle("POST /proxy", a.mutate(a.postProxySettings))
	mux.Handle("POST /proxy/apply-now", a.mutate(a.postProxyApplyNow))
	mux.Handle("POST /proxy/unfreeze", a.mutate(a.postUnfreeze))
	mux.Handle("POST /hostname", a.mutate(a.postHostname))
	mux.Handle("POST /reboot", a.mutate(a.postReboot))

	// JSON and streaming endpoints for the live panes.
	mux.Handle("GET /api/status", a.auth(a.apiStatus))
	mux.Handle("GET /api/interfaces", a.auth(a.apiInterfaces))
	mux.Handle("GET /api/wifi/scan", a.auth(a.apiWifiScan))
	mux.Handle("POST /api/proxy/test", a.auth(a.apiProxyTest))
	mux.Handle("GET /api/endpoints", a.auth(a.apiEndpoints))
	mux.Handle("GET /api/tunnel", a.auth(a.apiTunnel))
	mux.Handle("GET /api/ping", a.auth(a.apiPing))
	mux.Handle("GET /api/traceroute", a.auth(a.apiTraceroute))
	mux.Handle("GET /api/dns", a.auth(a.apiDNS))
	mux.Handle("GET /api/bundle", a.auth(a.apiBundle))
	mux.Handle("GET /api/activity/stream", a.auth(a.apiActivityStream))

	return securityHeaders(mux)
}

// handlerFunc is a handler that knows who is calling.
type handlerFunc func(http.ResponseWriter, *http.Request, session)

// auth requires a valid session, and forces a password change while the
// default password is still in place.
func (a *App) auth(next handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := a.readSession(r)
		if !ok {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "not authenticated", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login?next="+r.URL.Path, http.StatusSeeOther)
			return
		}
		// While the default password stands, only the settings page and the
		// password change itself are reachable.
		if a.store.Auth().MustChange &&
			r.URL.Path != "/settings" && r.URL.Path != "/password" && r.URL.Path != "/logout" {
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}
		next(w, r, s)
	})
}

// mutate wraps auth with CSRF validation and the read-only check.
func (a *App) mutate(next handlerFunc) http.Handler {
	return a.auth(func(w http.ResponseWriter, r *http.Request, s session) {
		if a.opts.ReadOnly {
			a.fail(w, r, http.StatusForbidden,
				"Read-only mode. Unset NETCFG_READONLY to make changes.")
			return
		}
		if err := r.ParseForm(); err != nil {
			a.fail(w, r, http.StatusBadRequest, "Could not read the submitted form.")
			return
		}
		if !a.checkCSRF(r, s) {
			a.fail(w, r, http.StatusForbidden, "Form expired. Reload and try again.")
			return
		}
		next(w, r, s)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// Everything is served from this origin and inlined nowhere, so the
		// policy can be strict without breaking the pages.
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; form-action 'self'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}
