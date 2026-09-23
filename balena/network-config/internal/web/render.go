package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// flashCookie carries a one-shot message across a redirect.
const flashCookie = "netcfgd_flash"

type flash struct {
	Kind string `json:"k"` // "ok" or "error"
	Text string `json:"t"`
}

func (a *App) setFlash(w http.ResponseWriter, kind, text string) {
	data, err := json.Marshal(flash{Kind: kind, Text: text})
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookie,
		Value:    base64.RawURLEncoding.EncodeToString(data),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30,
	})
}

func (a *App) takeFlash(w http.ResponseWriter, r *http.Request) *flash {
	c, err := r.Cookie(flashCookie)
	if err != nil {
		return nil
	}
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1})

	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return nil
	}
	var f flash
	if json.Unmarshal(raw, &f) != nil {
		return nil
	}
	return &f
}

// banner is a persistent warning shown on every page. These are the conditions
// an operator needs to know about regardless of which page they opened.
type banner struct {
	Kind string
	Text string
}

// pageData is the root of every template render.
type pageData struct {
	Title      string
	Nav        string
	CSRF       string
	User       string
	ReadOnly   bool
	Version    string
	MustChange bool
	Flash      *flash
	Banners    []banner
	Data       any
}

// page assembles the common page state, including the standing banners.
func (a *App) page(w http.ResponseWriter, r *http.Request, s session, nav, title string, data any) pageData {
	auth := a.store.Auth()
	p := pageData{
		Title:      title,
		Nav:        nav,
		CSRF:       a.csrfToken(s),
		User:       s.User,
		ReadOnly:   a.opts.ReadOnly,
		Version:    a.opts.Version,
		MustChange: auth.MustChange,
		Flash:      a.takeFlash(w, r),
		Data:       data,
	}

	if auth.MustChange {
		p.Banners = append(p.Banners, banner{"error",
			"This device is still using the default password."})
	}
	if a.nm == nil {
		p.Banners = append(p.Banners, banner{"error", fmt.Sprintf(
			"NetworkManager is not reachable, so interfaces cannot be configured: %v. "+
				"Needs the io.balena.features.dbus label and DBUS_SYSTEM_BUS_ADDRESS.", a.nmErr)})
	}
	if !a.sup.Available() {
		p.Banners = append(p.Banners, banner{"warn",
			"Supervisor API unavailable, so proxy settings cannot be applied. " +
				"Needs the io.balena.features.supervisor-api label."})
	}
	if b := a.ipv6Banner(r); b != nil {
		p.Banners = append(p.Banners, *b)
	}
	if st := a.state.State(); st.Frozen {
		p.Banners = append(p.Banners, banner{"error", "Watchdog stopped: " + st.FreezeReason})
	}
	if pc := a.currentPending(); pc != nil {
		p.Banners = append(p.Banners, banner{"warn", fmt.Sprintf(
			"A change to %s reverts in %s unless confirmed.",
			pc.Iface, time.Until(pc.Deadline).Round(time.Second))})
	}
	return p
}

// render writes a template, reporting failures rather than serving half a page.
func (a *App) render(w http.ResponseWriter, name string, p pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, name, p); err != nil {
		a.log.Errorf("web", "", "failed to render "+name, err.Error())
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// fail renders an error, as JSON for API calls and as a page otherwise.
func (a *App) fail(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	http.Error(w, msg, code)
}

// redirect sends the browser back with a flash message.
func (a *App) redirect(w http.ResponseWriter, r *http.Request, to, kind, msg string) {
	a.setFlash(w, kind, msg)
	http.Redirect(w, r, to, http.StatusSeeOther)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fsSub(f fs.FS, dir string) (fs.FS, error) { return fs.Sub(f, dir) }

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"since": func(t time.Time) string {
			if t.IsZero() {
				return "never"
			}
			return time.Since(t).Round(time.Second).String() + " ago"
		},
		"stamp": func(t time.Time) string {
			if t.IsZero() {
				return "—"
			}
			return t.Format("2006-01-02 15:04:05")
		},
		"yesno": func(b bool) string {
			if b {
				return "yes"
			}
			return "no"
		},
		"pct": func(n int32) string { return fmt.Sprintf("%d%%", n) },
		"inc": func(n int) int { return n + 1 },
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				if k, ok := kv[i].(string); ok {
					m[k] = kv[i+1]
				}
			}
			return m
		},
	}
}
