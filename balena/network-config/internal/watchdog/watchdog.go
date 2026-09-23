// Package watchdog keeps the host's proxy configuration in line with the
// operator's intent.
//
// The whole design is shaped by one fact: applying a proxy change through the
// balena Supervisor restarts balenaEngine on balenaOS 2.82.6 and newer, which
// restarts every container on the device — including this one. So the loop is
// built to do nothing by default. It reads the host's current configuration and
// compares before acting, rate-limits itself, records what it is about to do
// before doing it, and trips a breaker if it ever finds itself changing things
// repeatedly.
package watchdog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/normalframework/netcfgd/internal/activity"
	"github.com/normalframework/netcfgd/internal/proxyprobe"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
)

const source = "watchdog"

// freezeExpiry is how long the flap breaker holds before lifting on its own.
const freezeExpiry = time.Hour

// Prober runs a connection probe. Injected so tests need no network.
type Prober func(ctx context.Context, p *proxyprobe.Proxy, t proxyprobe.Target, timeout time.Duration) proxyprobe.Result

// Watchdog reconciles the host proxy configuration.
type Watchdog struct {
	cfg   *store.Store
	state *store.StateStore
	sup   supervisor.API
	log   *activity.Log

	// Injection points for tests.
	now    func() time.Time
	uptime func() (time.Duration, error)
	bootID func() string
	probe  Prober
}

// New builds a watchdog with production dependencies.
func New(cfg *store.Store, state *store.StateStore, sup supervisor.API, log *activity.Log) *Watchdog {
	return &Watchdog{
		cfg:    cfg,
		state:  state,
		sup:    sup,
		log:    log,
		now:    time.Now,
		uptime: store.Uptime,
		bootID: store.BootID,
		probe:  proxyprobe.Probe,
	}
}

// supervisorWait bounds how long startup waits for the Supervisor to answer.
const supervisorWait = 2 * time.Minute

// Run reconciles on an interval until the context is cancelled.
func (w *Watchdog) Run(ctx context.Context) {
	w.waitForSupervisor(ctx)

	// Reconcile once promptly at startup so a device that rebooted into a
	// broken proxy is not stuck for a full interval.
	w.ReconcileOnce(ctx)

	for {
		interval := time.Duration(w.cfg.Config().Watchdog.IntervalSeconds) * time.Second
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			w.ReconcileOnce(ctx)
		}
	}
}

// waitForSupervisor blocks until the Supervisor answers.
//
// On boot this container usually starts before the Supervisor is ready. Without
// this, the first pass fails and parks a 502 or a reset connection in the
// console as though something were wrong, when the only problem is that we
// asked too early. Bounded, and never fatal: if the Supervisor really is
// missing, the reconcile loop reports that on its own terms.
func (w *Watchdog) waitForSupervisor(ctx context.Context) {
	if !w.sup.Available() {
		return
	}
	if _, err := w.sup.GetProxy(ctx); err == nil {
		return
	}

	w.log.Infof(source, source, "waiting for the balena Supervisor to come up", "")
	deadline := w.now().Add(supervisorWait)
	for w.now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
		if _, err := w.sup.GetProxy(ctx); err == nil {
			w.log.Infof(source, source, "balena Supervisor is up", "")
			return
		}
	}
	w.log.Warnf(source, source, "the balena Supervisor did not come up",
		fmt.Sprintf("gave up after %s; proxy management will retry on the normal interval", supervisorWait))
}

// Outcome describes what one reconcile pass decided.
type Outcome struct {
	Action string `json:"action"` // "none", "set", "clear", "deferred", "frozen", "adopt"
	Reason string `json:"reason"`
	Error  string `json:"error,omitempty"`
}

// ReconcileOnce runs a single pass.
func (w *Watchdog) ReconcileOnce(ctx context.Context) Outcome {
	out := w.reconcile(ctx)
	w.state.Update(func(st *store.State) {
		st.LastRunAt = w.now()
		st.LastError = out.Error
	})
	return out
}

func (w *Watchdog) reconcile(ctx context.Context) Outcome {
	cfg := w.cfg.Config()
	now := w.now()

	if !w.sup.Available() {
		return Outcome{Action: "none", Reason: "supervisor API unavailable; proxy management is disabled"}
	}

	w.handleNewBoot()

	st := w.state.State()
	if st.Frozen {
		// A freeze must not be permanent: nobody can reach a device whose
		// network this watchdog has stopped managing. It lifts on its own
		// after freezeExpiry, which is long enough to break a flap loop and
		// short enough that a site recovers without a visit.
		if !st.FrozenAt.IsZero() && now.Sub(st.FrozenAt) >= freezeExpiry {
			w.state.Update(func(s *store.State) {
				s.Frozen = false
				s.FreezeReason = ""
				s.Changes = nil
			})
			w.log.Infof(source, source, "proxy watchdog breaker expired and has resumed",
				fmt.Sprintf("it was set %s ago", now.Sub(st.FrozenAt).Round(time.Minute)))
			st = w.state.State()
		} else {
			return Outcome{Action: "frozen", Reason: st.FreezeReason}
		}
	}

	// Recover from an interrupted change before deciding anything new.
	if st.Pending != nil {
		if o, done := w.recoverPending(ctx, *st.Pending, now); done {
			return o
		}
		st = w.state.State()
	}

	actual, err := w.sup.GetProxy(ctx)
	if err != nil {
		return Outcome{Action: "none", Reason: "could not read host proxy configuration", Error: err.Error()}
	}

	desired, reason, err := w.desired(ctx, cfg, st, actual)
	if err != nil {
		return Outcome{Action: "none", Reason: reason, Error: err.Error()}
	}

	want := toSupervisorProxy(desired)
	fp := fingerprint(desired)

	// The no-op guard. This is the single most important check in the package:
	// without it, every pass would restart every container on the device.
	if actual.SameAs(want) {
		st = w.state.State()
		if st.Applied == fp {
			return Outcome{Action: "none", Reason: reason}
		}
		// The host already matches but we have no record of applying it. If
		// there is no password, the comparison is complete and we can simply
		// adopt what is there. With a password we cannot tell — the Supervisor
		// never returns it — so one change is unavoidable.
		if desired.Password == "" {
			w.state.Update(func(s *store.State) { s.Applied = fp })
			w.log.Infof(source, source, "adopted the host's existing proxy configuration", reason)
			return Outcome{Action: "adopt", Reason: reason}
		}
	}

	// Rate limits.
	dwell := time.Duration(cfg.Watchdog.MinDwellMinutes) * time.Minute
	if !st.LastChangeAt.IsZero() && now.Sub(st.LastChangeAt) < dwell {
		wait := dwell - now.Sub(st.LastChangeAt)
		return Outcome{
			Action: "deferred",
			Reason: fmt.Sprintf("%s, but the last change was %s ago; waiting %s before changing again",
				reason, now.Sub(st.LastChangeAt).Round(time.Second), wait.Round(time.Second)),
		}
	}
	if n := w.state.ChangesInLastHour(now); n >= cfg.Watchdog.MaxChangesPerHour {
		msg := fmt.Sprintf("%d proxy changes in the last hour hit the limit of %d; no further changes will be made",
			n, cfg.Watchdog.MaxChangesPerHour)
		w.state.Update(func(s *store.State) {
			s.Frozen = true
			s.FreezeReason = msg
			s.FrozenAt = now
		})
		w.log.Errorf(source, source, "proxy watchdog frozen", msg)
		return Outcome{Action: "frozen", Reason: msg}
	}

	return w.apply(ctx, cfg, desired, want, fp, reason, now)
}

// apply records intent, calls the Supervisor, then records the result.
func (w *Watchdog) apply(ctx context.Context, cfg store.Config, desired store.ProxyConfig,
	want supervisor.Proxy, fp, reason string, now time.Time) Outcome {

	action := "clear"
	if want.Enabled() {
		action = "set"
	}

	// Write the intent first. The Supervisor call restarts this container, so
	// if we never get to record the result, the next start still knows a change
	// was attempted and applies the dwell before trying anything else.
	if err := w.state.Update(func(s *store.State) {
		s.Pending = &store.Intent{Action: action, Fingerprint: fp, At: now}
	}); err != nil {
		return Outcome{Action: "none", Reason: "could not record intent", Error: err.Error()}
	}

	w.log.Changef(source, source,
		fmt.Sprintf("applying proxy change (%s)", action),
		reason+" — restarts every container on this device")

	var err error
	if want.Enabled() {
		err = w.sup.SetProxy(ctx, want, cfg.Watchdog.Force)
	} else {
		err = w.sup.ClearProxy(ctx, cfg.Watchdog.Force)
	}
	if err != nil {
		// Leave Pending in place: the change may still have taken effect
		// before the connection dropped, and recovery reconciles it safely.
		w.log.Errorf(source, source, "proxy change failed", err.Error())
		return Outcome{Action: action, Reason: reason, Error: err.Error()}
	}

	w.state.RecordChange(now, fp)
	w.log.Changef(source, source, fmt.Sprintf("proxy %s", pastTense(action)), reason)

	// Reboot only when a proxy has been applied, never when one has been
	// cleared. The reason to reboot is that the Supervisor leaves existing
	// connections open, so a device already talking directly keeps doing so
	// until those connections break — that argument only applies to switching
	// traffic onto a proxy. Rebooting after a clear buys nothing, and in
	// fallback mode it creates a loop: apply, reboot, fail, clear, reboot,
	// apply again.
	if cfg.Watchdog.AutoReboot && want.Enabled() {
		w.log.Changef(source, source, "rebooting to apply the proxy change to all connections", "auto-reboot is enabled")
		if err := w.sup.Reboot(ctx, cfg.Watchdog.Force); err != nil {
			w.log.Errorf(source, source, "reboot request failed", err.Error())
		}
	}

	return Outcome{Action: action, Reason: reason}
}

// recoverPending resolves an intent that was recorded but whose outcome was
// never written, which happens whenever the engine restart kills us mid-change.
func (w *Watchdog) recoverPending(ctx context.Context, pending store.Intent, now time.Time) (Outcome, bool) {
	actual, err := w.sup.GetProxy(ctx)
	if err != nil {
		return Outcome{Action: "none", Reason: "recovering an interrupted proxy change", Error: err.Error()}, true
	}

	applied := (pending.Action == "set" && actual.Enabled()) || (pending.Action == "clear" && !actual.Enabled())
	if applied {
		w.state.RecordChange(pending.At, pending.Fingerprint)
		w.log.Infof(source, source, "recovered an interrupted proxy change",
			fmt.Sprintf("the %s took effect before this container restarted", pending.Action))
	} else {
		// The change did not land. Clear the intent but still count it against
		// the dwell, so a Supervisor that is failing does not get hammered.
		w.state.Update(func(s *store.State) {
			s.Pending = nil
			s.LastChangeAt = pending.At
		})
		w.log.Warnf(source, source, "an interrupted proxy change did not take effect",
			fmt.Sprintf("intent to %s was recorded at %s; the host does not reflect it", pending.Action, pending.At.Format(time.RFC3339)))
	}
	return Outcome{}, false
}

// handleNewBoot resets boot-scoped state when the host has rebooted.
func (w *Watchdog) handleNewBoot() {
	id := w.bootID()
	st := w.state.State()
	if st.BootID == id {
		return
	}
	wasFrozen := st.Frozen
	w.state.Update(func(s *store.State) {
		s.BootID = id
		// Fallback mode's decision is scoped to a boot: on a fresh boot it
		// gets to try the proxy again.
		s.FallbackOutcome = store.FallbackPending
		s.Passes = 0
		s.Failures = 0
		// The change history and the breaker deliberately survive a reboot.
		// Clearing them here would make a reboot loop undetectable, which is
		// exactly the failure the breaker exists to stop: every cycle would
		// start with a clean slate and the count would never reach the limit.
		// A frozen watchdog recovers on its own via freezeExpiry instead.
	})
	detail := "boot " + id
	if wasFrozen {
		detail += "; the flap breaker is still set from before the reboot"
	}
	if st.BootID != "" {
		w.log.Infof(source, source, "host reboot detected; watchdog state reset", detail)
	}
}

// desired computes the proxy configuration this mode wants right now.
func (w *Watchdog) desired(ctx context.Context, cfg store.Config, st store.State, actual supervisor.Proxy) (store.ProxyConfig, string, error) {
	none := store.ProxyConfig{Mode: cfg.Proxy.Mode}

	switch cfg.Proxy.Mode {
	case store.ModeOff:
		return none, "mode is off", nil

	case store.ModeStatic:
		return cfg.Proxy, "mode is static", nil

	case store.ModeFallback:
		return w.desiredFallback(ctx, cfg, st, actual)

	case store.ModeDynamic:
		return w.desiredDynamic(ctx, cfg, st, actual)
	}
	return none, "unknown mode", fmt.Errorf("unknown watchdog mode %q", cfg.Proxy.Mode)
}

// desiredFallback applies the proxy at boot and gives up after the grace
// period if the device still cannot reach the internet.
func (w *Watchdog) desiredFallback(ctx context.Context, cfg store.Config, st store.State, actual supervisor.Proxy) (store.ProxyConfig, string, error) {
	switch st.FallbackOutcome {
	case store.FallbackConfirmed:
		return cfg.Proxy, "fallback: proxy confirmed this boot", nil
	case store.FallbackCleared:
		return store.ProxyConfig{Mode: cfg.Proxy.Mode},
			"fallback: cleared this boot, the proxy did not work in time", nil
	}

	// Apply the proxy before judging it. The probe below reads the effective
	// path, which only means "through the proxy" once the proxy is actually in
	// force. Running it first measures the unproxied route, and on a device
	// whose direct path works that latches a broken proxy as confirmed — after
	// which fallback can never clear it, which is the one thing it exists for.
	want := toSupervisorProxy(cfg.Proxy)
	if want.Enabled() && !actual.SameAs(want) {
		return cfg.Proxy, "fallback: applying the proxy before testing it", nil
	}

	// The proxy is in force now, so a direct probe from this container follows
	// it — redsocks intercepts transparently.
	res := w.probeTarget(ctx, nil, cfg.Watchdog.ProbeTarget, true)
	w.state.Update(func(s *store.State) { s.LastProbeOK = res.OK })

	if res.OK {
		w.state.Update(func(s *store.State) { s.FallbackOutcome = store.FallbackConfirmed })
		w.log.Infof(source, source, "fallback: connectivity confirmed", res.Summary())
		return cfg.Proxy, "fallback: connectivity confirmed through the proxy", nil
	}

	uptime, err := w.uptime()
	if err != nil {
		return cfg.Proxy, "fallback: no host uptime, keeping the proxy", nil
	}
	grace := time.Duration(cfg.Watchdog.GraceMinutes) * time.Minute
	if uptime < grace {
		return cfg.Proxy, fmt.Sprintf(
			"fallback: within the %s grace period (uptime %s)", grace, uptime.Round(time.Second)), nil
	}

	w.state.Update(func(s *store.State) { s.FallbackOutcome = store.FallbackCleared })
	w.log.Warnf(source, source, "fallback: clearing the proxy",
		fmt.Sprintf("no connectivity %s after boot: %s", uptime.Round(time.Second), res.Summary()))
	return store.ProxyConfig{Mode: cfg.Proxy.Mode},
		fmt.Sprintf("fallback: no connectivity %s after boot, clearing the proxy", grace), nil
}

// desiredDynamic only applies a proxy that has been proven to work, and only
// removes one once the direct path has been proven to work instead.
func (w *Watchdog) desiredDynamic(ctx context.Context, cfg store.Config, st store.State, actual supervisor.Proxy) (store.ProxyConfig, string, error) {
	none := store.ProxyConfig{Mode: cfg.Proxy.Mode}

	candidate := &proxyprobe.Proxy{
		Type: cfg.Proxy.Type, IP: cfg.Proxy.IP, Port: cfg.Proxy.Port,
		Login: cfg.Proxy.Login, Password: cfg.Proxy.Password,
	}
	if cfg.Proxy.IP == "" || cfg.Proxy.Port == 0 {
		return none, "dynamic: nothing configured to test", nil
	}

	res := w.probeTarget(ctx, candidate, cfg.Watchdog.ProbeTarget, true)

	var passes, failures int
	w.state.Update(func(s *store.State) {
		if res.OK {
			s.Passes++
			s.Failures = 0
		} else {
			s.Failures++
			s.Passes = 0
		}
		s.LastProbeOK = res.OK
		passes, failures = s.Passes, s.Failures
	})

	if res.OK {
		if passes >= cfg.Watchdog.RequiredPasses {
			return cfg.Proxy, fmt.Sprintf(
				"dynamic: the proxy passed %d consecutive probes (%s)", passes, res.Summary()), nil
		}
		return currentIntent(cfg, actual), fmt.Sprintf(
			"dynamic: %d of %d passes needed before applying", passes, cfg.Watchdog.RequiredPasses), nil
	}

	if failures >= cfg.Watchdog.RequiredFailures {
		// Only remove a proxy if the direct path actually works. Removing it
		// when nothing works would take a device that is merely slow and leave
		// it with no route out at all.
		// Whether the direct path works is a question about the unproxied
		// route, so this one is deliberately not pinned to IPv4.
		direct := w.probeTarget(ctx, nil, cfg.Watchdog.ProbeTarget, false)
		if direct.OK {
			return none, fmt.Sprintf(
				"dynamic: the proxy failed %d consecutive probes and the direct path works (%s)",
				failures, res.Summary()), nil
		}
		return currentIntent(cfg, actual), fmt.Sprintf(
			"dynamic: proxy failed %d times but the direct path is down too, so nothing changed (%s)",
			failures, res.Summary()), nil
	}

	return currentIntent(cfg, actual), fmt.Sprintf(
		"dynamic: %d of %d failures needed before removing (%s)",
		failures, cfg.Watchdog.RequiredFailures, res.Summary()), nil
}

// currentIntent keeps whatever the host already has, so an undecided pass never
// causes a change.
func currentIntent(cfg store.Config, actual supervisor.Proxy) store.ProxyConfig {
	if actual.Enabled() {
		return cfg.Proxy
	}
	return store.ProxyConfig{Mode: cfg.Proxy.Mode}
}

// probeTarget runs a probe against the configured target address.
//
// Probes are pinned to IPv4 whenever a proxy is in play. balenaOS's transparent
// redirector installs iptables rules only — there is no ip6tables equivalent —
// so on a dual-stack network IPv6 traffic ignores the proxy entirely. Without
// this pin, a probe resolves an IPv6 address, sails past a completely dead
// proxy, and reports the path as healthy.
func (w *Watchdog) probeTarget(ctx context.Context, p *proxyprobe.Proxy, addr string, proxyInPlay bool) proxyprobe.Result {
	kind := proxyprobe.KindHTTPS
	if !strings.HasSuffix(addr, ":443") {
		kind = proxyprobe.KindTCP
	}
	network := "tcp"
	if proxyInPlay {
		network = "tcp4"
	}
	return w.probe(ctx, p, proxyprobe.Target{
		Name: "watchdog", Addr: addr, Kind: kind, Network: network,
	}, proxyprobe.DefaultTimeout)
}

// Unfreeze clears the flap breaker after an operator has looked at it.
func (w *Watchdog) Unfreeze(actor string) error {
	err := w.state.Update(func(s *store.State) {
		s.Frozen = false
		s.FreezeReason = ""
		s.Changes = nil
	})
	if err == nil {
		w.log.Infof(source, actor, "proxy watchdog breaker cleared", "")
	}
	return err
}

// toSupervisorProxy converts operator intent to the Supervisor's shape. A
// disabled configuration becomes the zero Proxy, which clears the host setting.
func toSupervisorProxy(c store.ProxyConfig) supervisor.Proxy {
	if c.Mode == store.ModeOff || c.IP == "" || c.Port == 0 || c.Type == "" {
		return supervisor.Proxy{}
	}
	p := supervisor.Proxy{
		Type:     supervisor.ProxyType(c.Type),
		IP:       c.IP,
		Port:     supervisor.Port(c.Port),
		Login:    c.Login,
		Password: c.Password,
		NoProxy:  c.NoProxy,
	}
	switch strings.TrimSpace(c.DNS) {
	case "":
	case "on", "true":
		p.DNS = true
	default:
		p.DNS = c.DNS
	}
	return p
}

// fingerprint hashes everything that defines a proxy configuration, including
// the password, so a password-only change is still detected as a change.
func fingerprint(c store.ProxyConfig) string {
	if c.Mode == store.ModeOff || c.IP == "" || c.Port == 0 {
		return "none"
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%d|%s|%s|%s|%s",
		c.Type, c.IP, c.Port, c.Login, c.Password, strings.Join(c.NoProxy, ","), c.DNS)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func pastTense(action string) string {
	if action == "set" {
		return "applied"
	}
	return "cleared"
}
