package web

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/normalframework/netcfgd/internal/netmgr"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
)

func (a *App) requireNM(w http.ResponseWriter, r *http.Request) bool {
	if a.nm == nil {
		a.redirect(w, r, "/interfaces", "error",
			"NetworkManager is not reachable, so interfaces cannot be changed.")
		return false
	}
	return true
}

// postIPv4 sets an interface to DHCP or a static address.
//
// The change is wrapped in the rollback guard, because this is the operation
// that can take the device off the network and strand an installer.
func (a *App) postIPv4(w http.ResponseWriter, r *http.Request, s session) {
	if !a.requireNM(w, r) {
		return
	}
	iface := r.PathValue("iface")
	snap := a.nm.Snapshot(iface)

	var err error
	var summary string
	switch r.FormValue("method") {
	case "auto":
		err = a.nm.SetDHCP(iface)
		summary = "DHCP"
	case "manual":
		dns, dnsErr := netmgr.ParseDNSList(r.FormValue("dns"))
		if dnsErr != nil {
			a.redirect(w, r, "/interfaces", "error", dnsErr.Error())
			return
		}
		cfg := netmgr.StaticConfig{
			Address: strings.TrimSpace(r.FormValue("address")),
			Netmask: strings.TrimSpace(r.FormValue("netmask")),
			Gateway: strings.TrimSpace(r.FormValue("gateway")),
			DNS:     dns,
		}
		err = a.nm.SetStatic(iface, cfg)
		summary = fmt.Sprintf("static %s/%s gw %s", cfg.Address, cfg.Netmask, cfg.Gateway)
	default:
		a.redirect(w, r, "/interfaces", "error", "Choose either automatic or static addressing.")
		return
	}
	if err != nil {
		a.log.Errorf("web", s.User, "failed to configure "+iface, err.Error())
		a.redirect(w, r, "/interfaces", "error", err.Error())
		return
	}

	a.log.Changef("web", s.User, fmt.Sprintf("configured %s for %s", iface, summary),
		"awaiting confirmation")

	if _, err := a.armPending(iface, snap); err != nil {
		a.log.Errorf("web", s.User, "could not arm the rollback guard", err.Error())
	}
	a.redirect(w, r, "/interfaces", "ok", fmt.Sprintf(
		"%s set to %s. Confirm below within %s or it reverts.", iface, summary, confirmWindow))
}

func (a *App) postIfaceState(w http.ResponseWriter, r *http.Request, s session) {
	if !a.requireNM(w, r) {
		return
	}
	iface := r.PathValue("iface")
	enabled := r.FormValue("enabled") == "true"

	if err := a.nm.SetEnabled(iface, enabled); err != nil {
		a.redirect(w, r, "/interfaces", "error", err.Error())
		return
	}
	verb := "disabled"
	if enabled {
		verb = "enabled"
	}
	a.log.Changef("web", s.User, fmt.Sprintf("%s %s", verb, iface), "")
	a.redirect(w, r, "/interfaces", "ok", fmt.Sprintf("%s %s.", iface, verb))
}

func (a *App) postWifi(w http.ResponseWriter, r *http.Request, s session) {
	if !a.requireNM(w, r) {
		return
	}
	iface := r.PathValue("iface")
	ssid := strings.TrimSpace(r.FormValue("ssid"))
	snap := a.nm.Snapshot(iface)

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	if err := a.nm.SetWifi(ctx, iface, ssid, r.FormValue("psk")); err != nil {
		a.log.Errorf("web", s.User, "failed to join "+ssid, err.Error())
		a.redirect(w, r, "/interfaces", "error", err.Error())
		return
	}

	a.log.Changef("web", s.User, fmt.Sprintf("joined %s on %s", ssid, iface), "")
	if _, err := a.armPending(iface, snap); err != nil {
		a.log.Errorf("web", s.User, "could not arm the rollback guard", err.Error())
	}
	a.redirect(w, r, "/interfaces", "ok", fmt.Sprintf(
		"Joined %s. Confirm within %s or it reverts.", ssid, confirmWindow))
}

// postPriority reorders interfaces for failover.
func (a *App) postPriority(w http.ResponseWriter, r *http.Request, s session) {
	if !a.requireNM(w, r) {
		return
	}
	var ranks []netmgr.Rank
	for key, values := range r.Form {
		name, ok := strings.CutPrefix(key, "rank-")
		if !ok || len(values) == 0 {
			continue
		}
		n, err := strconv.Atoi(values[0])
		if err != nil {
			a.redirect(w, r, "/interfaces", "error",
				fmt.Sprintf("%q is not a valid rank for %s.", values[0], name))
			return
		}
		ranks = append(ranks, netmgr.Rank{Iface: name, Rank: n})
	}
	if len(ranks) == 0 {
		a.redirect(w, r, "/interfaces", "error", "No interface ranks were submitted.")
		return
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i].Rank < ranks[j].Rank })

	if err := a.nm.SetPriority(ranks); err != nil {
		a.redirect(w, r, "/interfaces", "error", err.Error())
		return
	}

	var order []string
	for _, rk := range ranks {
		order = append(order, rk.Iface)
	}
	a.log.Changef("web", s.User, "set interface priority", strings.Join(order, " > "))
	a.redirect(w, r, "/interfaces", "ok", "Interface priority saved: "+strings.Join(order, " then "))
}

func (a *App) postConfirm(w http.ResponseWriter, r *http.Request, s session) {
	if err := a.confirmPending(r.FormValue("token")); err != nil {
		a.redirect(w, r, "/interfaces", "error", err.Error())
		return
	}
	a.redirect(w, r, "/interfaces", "ok", "Change confirmed and kept.")
}

func (a *App) postRevert(w http.ResponseWriter, r *http.Request, s session) {
	if err := a.revertPending(); err != nil {
		a.redirect(w, r, "/interfaces", "error", err.Error())
		return
	}
	a.redirect(w, r, "/interfaces", "ok", "Change reverted.")
}

// postProxySettings saves proxy intent. It does not itself call the Supervisor:
// the watchdog owns that, so there is exactly one place that decides when the
// device's containers get restarted.
func (a *App) postProxySettings(w http.ResponseWriter, r *http.Request, s session) {
	mode := store.Mode(r.FormValue("mode"))
	if !mode.Valid() {
		a.redirect(w, r, "/proxy", "error", "Choose a valid watchdog mode.")
		return
	}

	port := 0
	if v := strings.TrimSpace(r.FormValue("port")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 65535 {
			a.redirect(w, r, "/proxy", "error", fmt.Sprintf("%q is not a valid port.", v))
			return
		}
		port = n
	}

	proxyType := strings.TrimSpace(r.FormValue("type"))
	if mode != store.ModeOff && !supervisor.ProxyType(proxyType).Valid() {
		a.redirect(w, r, "/proxy", "error", "Choose a valid proxy type.")
		return
	}

	noProxy := splitList(r.FormValue("noProxy"))

	err := a.store.Update(func(c *store.Config) {
		c.Proxy.Mode = mode
		c.Proxy.Type = proxyType
		c.Proxy.IP = strings.TrimSpace(r.FormValue("ip"))
		c.Proxy.Port = port
		c.Proxy.Login = strings.TrimSpace(r.FormValue("login"))
		c.Proxy.NoProxy = noProxy
		c.Proxy.DNS = strings.TrimSpace(r.FormValue("dns"))
		// An empty password field leaves the stored password alone, so saving
		// an unrelated setting does not silently wipe it.
		if pw := r.FormValue("password"); pw != "" {
			c.Proxy.Password = pw
		}
		if r.FormValue("clearPassword") == "true" {
			c.Proxy.Password = ""
		}

		c.Watchdog.GraceMinutes = atoiOr(r.FormValue("graceMinutes"), c.Watchdog.GraceMinutes)
		c.Watchdog.RequiredPasses = atoiOr(r.FormValue("requiredPasses"), c.Watchdog.RequiredPasses)
		c.Watchdog.RequiredFailures = atoiOr(r.FormValue("requiredFailures"), c.Watchdog.RequiredFailures)
		c.Watchdog.MinDwellMinutes = atoiOr(r.FormValue("minDwellMinutes"), c.Watchdog.MinDwellMinutes)
		c.Watchdog.MaxChangesPerHour = atoiOr(r.FormValue("maxChangesPerHour"), c.Watchdog.MaxChangesPerHour)
		c.Watchdog.IntervalSeconds = atoiOr(r.FormValue("intervalSeconds"), c.Watchdog.IntervalSeconds)
		if t := strings.TrimSpace(r.FormValue("probeTarget")); t != "" {
			c.Watchdog.ProbeTarget = t
		}
		c.Watchdog.AutoReboot = r.FormValue("autoReboot") == "true"
		c.Watchdog.Force = r.FormValue("force") == "true"
	})
	if err != nil {
		a.redirect(w, r, "/proxy", "error", "Could not save: "+err.Error())
		return
	}

	a.log.Changef("web", s.User, "proxy settings saved",
		fmt.Sprintf("mode=%s type=%s addr=%s:%d", mode, proxyType, r.FormValue("ip"), port))
	a.redirect(w, r, "/proxy", "ok", "Saved. Applied on the next watchdog pass, or run it now.")
}

// postProxyApplyNow runs one watchdog pass immediately.
func (a *App) postProxyApplyNow(w http.ResponseWriter, r *http.Request, s session) {
	a.log.Infof("web", s.User, "manual watchdog run requested", "")
	out := a.dog.ReconcileOnce(r.Context())

	kind := "ok"
	msg := out.Reason
	switch {
	case out.Error != "":
		kind, msg = "error", out.Reason+": "+out.Error
	case out.Action == "set" || out.Action == "clear":
		msg = "Applied — containers are restarting. " + out.Reason
	case out.Action == "none":
		msg = "No change needed. " + out.Reason
	}
	a.redirect(w, r, "/proxy", kind, msg)
}

func (a *App) postUnfreeze(w http.ResponseWriter, r *http.Request, s session) {
	if err := a.dog.Unfreeze(s.User); err != nil {
		a.redirect(w, r, "/proxy", "error", err.Error())
		return
	}
	a.redirect(w, r, "/proxy", "ok", "Breaker cleared.")
}

func (a *App) postHostname(w http.ResponseWriter, r *http.Request, s session) {
	hostname := strings.TrimSpace(r.FormValue("hostname"))
	if hostname == "" {
		a.redirect(w, r, "/settings", "error", "Enter a hostname.")
		return
	}
	if err := a.sup.SetHostname(r.Context(), hostname, false); err != nil {
		a.redirect(w, r, "/settings", "error", err.Error())
		return
	}
	a.log.Changef("web", s.User, "hostname set to "+hostname, "applies to containers after a reboot")
	a.redirect(w, r, "/settings", "ok", "Saved. Containers keep the old hostname until reboot.")
}

func (a *App) postReboot(w http.ResponseWriter, r *http.Request, s session) {
	a.log.Changef("web", s.User, "reboot requested", "")
	if err := a.sup.Reboot(r.Context(), r.FormValue("force") == "true"); err != nil {
		a.redirect(w, r, "/settings", "error", err.Error())
		return
	}
	a.redirect(w, r, "/settings", "ok", "Rebooting. Unreachable for a minute or two.")
}

func splitList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t' || r == ';'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func atoiOr(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}
