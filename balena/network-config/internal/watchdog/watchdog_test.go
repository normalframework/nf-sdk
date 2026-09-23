package watchdog

import (
	"context"
	"testing"
	"time"

	"github.com/normalframework/netcfgd/internal/activity"
	"github.com/normalframework/netcfgd/internal/proxyprobe"
	"github.com/normalframework/netcfgd/internal/store"
	"github.com/normalframework/netcfgd/internal/supervisor"
)

// fakeSupervisor records calls instead of restarting a device.
type fakeSupervisor struct {
	proxy   supervisor.Proxy
	sets    int
	clears  int
	reboots int
	getErr  error
	setErr  error
}

func (f *fakeSupervisor) Available() bool { return true }

func (f *fakeSupervisor) GetProxy(context.Context) (supervisor.Proxy, error) {
	return f.proxy, f.getErr
}

func (f *fakeSupervisor) SetProxy(_ context.Context, p supervisor.Proxy, _ bool) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.sets++
	// Mirror the real client, which stores the normalized form.
	f.proxy = p.Normalized()
	return nil
}

func (f *fakeSupervisor) ClearProxy(context.Context, bool) error {
	f.clears++
	f.proxy = supervisor.Proxy{}
	return nil
}

func (f *fakeSupervisor) Reboot(context.Context, bool) error {
	f.reboots++
	return nil
}

// harness wires a watchdog to fakes with a controllable clock.
type harness struct {
	w     *Watchdog
	sup   *fakeSupervisor
	cfg   *store.Store
	state *store.StateStore
	now   time.Time
	up    time.Duration
	boot  string
	probe func() proxyprobe.Result
}

func newHarness(t *testing.T, proxyCfg store.ProxyConfig) *harness {
	t.Helper()
	dir := t.TempDir()

	cfg, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open config store: %v", err)
	}
	if err := cfg.Update(func(c *store.Config) { c.Proxy = proxyCfg }); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	state, err := store.OpenState(dir)
	if err != nil {
		t.Fatalf("open state store: %v", err)
	}

	h := &harness{
		sup:   &fakeSupervisor{},
		cfg:   cfg,
		state: state,
		now:   time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		up:    time.Hour,
		boot:  "boot-1",
		probe: func() proxyprobe.Result { return proxyprobe.Result{OK: true} },
	}
	h.w = New(cfg, state, h.sup, activity.New(50))
	h.w.now = func() time.Time { return h.now }
	h.w.uptime = func() (time.Duration, error) { return h.up, nil }
	h.w.bootID = func() string { return h.boot }
	h.w.probe = func(context.Context, *proxyprobe.Proxy, proxyprobe.Target, time.Duration) proxyprobe.Result {
		return h.probe()
	}
	return h
}

func (h *harness) run(t *testing.T) Outcome {
	t.Helper()
	return h.w.ReconcileOnce(context.Background())
}

func staticProxy() store.ProxyConfig {
	return store.ProxyConfig{
		Mode: store.ModeStatic, Type: "http-connect", IP: "10.0.0.9", Port: 8080,
	}
}

// The no-op guard is the most important behaviour in the package: a redundant
// PATCH restarts balenaEngine and therefore every container on the device.
func TestStaticModeAppliesOnceThenStopsChanging(t *testing.T) {
	h := newHarness(t, staticProxy())

	if got := h.run(t); got.Action != "set" {
		t.Fatalf("first pass: got action %q (%s), want set", got.Action, got.Reason)
	}
	if h.sup.sets != 1 {
		t.Fatalf("first pass: got %d SetProxy calls, want 1", h.sup.sets)
	}

	// Many passes over a long period must not touch the Supervisor again.
	for i := 0; i < 20; i++ {
		h.now = h.now.Add(30 * time.Minute)
		if got := h.run(t); got.Action != "none" {
			t.Fatalf("pass %d: got action %q (%s), want none", i, got.Action, got.Reason)
		}
	}
	if h.sup.sets != 1 {
		t.Fatalf("got %d SetProxy calls after 20 further passes, want 1", h.sup.sets)
	}
}

// A device whose host already carries the right passwordless proxy should be
// adopted rather than re-applied, so redeploying netcfgd does not restart
// every container.
func TestAdoptsMatchingHostConfigWithoutChanging(t *testing.T) {
	h := newHarness(t, staticProxy())
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080}.Normalized()

	got := h.run(t)
	if got.Action != "adopt" {
		t.Fatalf("got action %q (%s), want adopt", got.Action, got.Reason)
	}
	if h.sup.sets != 0 || h.sup.clears != 0 {
		t.Fatalf("got %d sets and %d clears, want none", h.sup.sets, h.sup.clears)
	}
}

// With a password the Supervisor never echoes it back, so equality cannot be
// established and exactly one change is expected — but only one.
func TestPasswordedProxyAppliesExactlyOnce(t *testing.T) {
	cfg := staticProxy()
	cfg.Login, cfg.Password = "user", "secret"
	h := newHarness(t, cfg)
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080, Login: "user"}.Normalized()

	if got := h.run(t); got.Action != "set" {
		t.Fatalf("first pass: got %q (%s), want set", got.Action, got.Reason)
	}
	h.now = h.now.Add(time.Hour)
	if got := h.run(t); got.Action != "none" {
		t.Fatalf("second pass: got %q (%s), want none", got.Action, got.Reason)
	}
	if h.sup.sets != 1 {
		t.Fatalf("got %d SetProxy calls, want 1", h.sup.sets)
	}
}

func TestDwellDefersRapidChanges(t *testing.T) {
	h := newHarness(t, staticProxy())
	h.run(t) // applies

	// Change the configuration immediately; the dwell must hold it back.
	h.cfg.Update(func(c *store.Config) { c.Proxy.IP = "10.0.0.10" })
	h.now = h.now.Add(time.Minute)

	got := h.run(t)
	if got.Action != "deferred" {
		t.Fatalf("got action %q (%s), want deferred", got.Action, got.Reason)
	}
	if h.sup.sets != 1 {
		t.Fatalf("got %d SetProxy calls, want 1", h.sup.sets)
	}

	// Past the dwell, the change goes through.
	h.now = h.now.Add(time.Duration(store.DefaultWatchdog().MinDwellMinutes) * time.Minute)
	if got := h.run(t); got.Action != "set" {
		t.Fatalf("after dwell: got %q (%s), want set", got.Action, got.Reason)
	}
}

func TestFlapBreakerFreezesAfterTooManyChanges(t *testing.T) {
	h := newHarness(t, staticProxy())
	max := store.DefaultWatchdog().MaxChangesPerHour
	dwell := time.Duration(store.DefaultWatchdog().MinDwellMinutes) * time.Minute

	for i := 0; i < max; i++ {
		h.cfg.Update(func(c *store.Config) { c.Proxy.Port = 8080 + i })
		if got := h.run(t); got.Action != "set" {
			t.Fatalf("change %d: got %q (%s), want set", i, got.Action, got.Reason)
		}
		h.now = h.now.Add(dwell)
	}

	h.cfg.Update(func(c *store.Config) { c.Proxy.Port = 9999 })
	got := h.run(t)
	if got.Action != "frozen" {
		t.Fatalf("got action %q (%s), want frozen", got.Action, got.Reason)
	}
	if h.sup.sets != max {
		t.Fatalf("got %d SetProxy calls, want %d", h.sup.sets, max)
	}

	// It stays frozen while the freeze is current.
	h.now = h.now.Add(30 * time.Minute)
	if got := h.run(t); got.Action != "frozen" {
		t.Fatalf("30 minutes after freezing: got %q, want frozen", got.Action)
	}
	if err := h.w.Unfreeze("tester"); err != nil {
		t.Fatalf("unfreeze: %v", err)
	}
	if got := h.run(t); got.Action != "set" {
		t.Fatalf("after unfreeze: got %q (%s), want set", got.Action, got.Reason)
	}
}

// A freeze must lift on its own. Nobody can reach a device whose networking
// this watchdog has stopped managing, so a permanent freeze would need a site
// visit to clear.
func TestBreakerExpiresOnItsOwn(t *testing.T) {
	h := newHarness(t, staticProxy())
	h.state.Update(func(s *store.State) {
		s.BootID = h.boot
		s.Frozen = true
		s.FreezeReason = "test"
		s.FrozenAt = h.now
	})

	h.now = h.now.Add(freezeExpiry - time.Minute)
	if got := h.run(t); got.Action != "frozen" {
		t.Fatalf("just before expiry: got %q, want frozen", got.Action)
	}

	h.now = h.now.Add(2 * time.Minute)
	if got := h.run(t); got.Action == "frozen" {
		t.Fatalf("after expiry the breaker should have lifted, got %q", got.Action)
	}
}

// The breaker must survive a reboot. If a reboot reset it, a device caught in
// an apply/reboot/clear/reboot loop would start every cycle with a clean count
// and the breaker could never trip — which is the one situation it exists for.
func TestBreakerSurvivesReboot(t *testing.T) {
	h := newHarness(t, staticProxy())
	h.state.Update(func(s *store.State) {
		s.BootID = "boot-1"
		s.Frozen = true
		s.FreezeReason = "test"
		s.FrozenAt = h.now
	})

	h.boot = "boot-2"
	if got := h.run(t); got.Action != "frozen" {
		t.Fatalf("breaker did not survive the reboot: got %q (%s)", got.Action, got.Reason)
	}
	if h.state.State().Changes == nil && h.sup.sets > 0 {
		t.Fatal("change history was reset by the reboot")
	}
}

// Auto-reboot after clearing a proxy buys nothing and, in fallback mode, spins
// the device: apply, reboot, fail, clear, reboot, apply again.
func TestAutoRebootDoesNotFireWhenClearingAProxy(t *testing.T) {
	h := newHarness(t, store.ProxyConfig{Mode: store.ModeOff})
	h.cfg.Update(func(c *store.Config) { c.Watchdog.AutoReboot = true })
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080}.Normalized()

	if got := h.run(t); got.Action != "clear" {
		t.Fatalf("got %q (%s), want clear", got.Action, got.Reason)
	}
	if h.sup.reboots != 0 {
		t.Fatalf("got %d reboots after clearing a proxy, want 0", h.sup.reboots)
	}
}

// The engine restart caused by a change can kill this process mid-flight. The
// next start must not repeat the change.
func TestRecoversInterruptedChangeWithoutRepeatingIt(t *testing.T) {
	h := newHarness(t, staticProxy())

	// Simulate: intent recorded, Supervisor applied it, then we were killed.
	h.state.Update(func(s *store.State) {
		s.BootID = "boot-1"
		s.Pending = &store.Intent{Action: "set", Fingerprint: fingerprint(h.cfg.Config().Proxy), At: h.now}
	})
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080}.Normalized()

	if got := h.run(t); got.Action != "none" {
		t.Fatalf("got action %q (%s), want none", got.Action, got.Reason)
	}
	if h.sup.sets != 0 {
		t.Fatalf("got %d SetProxy calls, want 0 — the interrupted change was repeated", h.sup.sets)
	}
	if h.state.State().Pending != nil {
		t.Fatal("pending intent was not cleared after recovery")
	}
}

// Fallback must not judge the path until the proxy is actually applied.
// Probing first measures the unproxied route, and a working direct path would
// latch a broken proxy as "confirmed" — permanently disabling the very rescue
// this mode provides.
func TestFallbackDoesNotConfirmBeforeTheProxyIsApplied(t *testing.T) {
	cfg := staticProxy()
	cfg.Mode = store.ModeFallback
	h := newHarness(t, cfg)
	h.up = time.Minute

	// The direct path works; the proxy is not applied yet.
	h.probe = func() proxyprobe.Result { return proxyprobe.Result{OK: true} }

	if got := h.run(t); got.Action != "set" {
		t.Fatalf("first pass: got %q (%s), want set", got.Action, got.Reason)
	}
	if outcome := h.state.State().FallbackOutcome; outcome != store.FallbackPending {
		t.Fatalf("latched %q before the proxy was applied; it must stay undecided", outcome)
	}

	// With the proxy now in force and failing, fallback must still be able to
	// clear it once the grace period expires.
	h.probe = func() proxyprobe.Result { return proxyprobe.Result{OK: false, Failed: "tls"} }
	h.up = time.Duration(store.DefaultWatchdog().GraceMinutes)*time.Minute + time.Minute
	h.now = h.now.Add(time.Duration(store.DefaultWatchdog().MinDwellMinutes) * time.Minute)

	if got := h.run(t); got.Action != "clear" {
		t.Fatalf("after grace: got %q (%s), want clear", got.Action, got.Reason)
	}
}

func TestFallbackKeepsProxyWhileConnectivityWorks(t *testing.T) {
	cfg := staticProxy()
	cfg.Mode = store.ModeFallback
	h := newHarness(t, cfg)
	h.up = time.Minute
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080}.Normalized()

	if got := h.run(t); got.Action == "clear" {
		t.Fatalf("cleared a working proxy: %s", got.Reason)
	}
	if h.state.State().FallbackOutcome != store.FallbackConfirmed {
		t.Fatalf("got outcome %q, want confirmed", h.state.State().FallbackOutcome)
	}
}

func TestFallbackClearsProxyAfterGracePeriod(t *testing.T) {
	cfg := staticProxy()
	cfg.Mode = store.ModeFallback
	h := newHarness(t, cfg)
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080}.Normalized()
	h.probe = func() proxyprobe.Result { return proxyprobe.Result{OK: false, Failed: "tcp"} }

	// Inside the grace period the proxy stays applied.
	h.up = 2 * time.Minute
	if got := h.run(t); got.Action == "clear" {
		t.Fatalf("within grace: cleared the proxy early (%s)", got.Reason)
	}

	// Past it, the proxy is cleared.
	h.up = time.Duration(store.DefaultWatchdog().GraceMinutes)*time.Minute + time.Minute
	h.now = h.now.Add(time.Duration(store.DefaultWatchdog().MinDwellMinutes) * time.Minute)
	if got := h.run(t); got.Action != "clear" {
		t.Fatalf("past grace: got %q (%s), want clear", got.Action, got.Reason)
	}
	if h.sup.clears != 1 {
		t.Fatalf("got %d ClearProxy calls, want 1", h.sup.clears)
	}

	// The decision is latched for this boot: no further churn.
	h.now = h.now.Add(time.Hour)
	if got := h.run(t); got.Action != "none" {
		t.Fatalf("after clearing: got %q (%s), want none", got.Action, got.Reason)
	}
}

func TestDynamicWaitsForConsecutivePassesBeforeApplying(t *testing.T) {
	cfg := staticProxy()
	cfg.Mode = store.ModeDynamic
	h := newHarness(t, cfg)

	required := store.DefaultWatchdog().RequiredPasses
	for i := 1; i < required; i++ {
		got := h.run(t)
		if got.Action == "set" {
			t.Fatalf("probe %d: proxy applied after only %d of %d passes (%s)", i, i, required, got.Reason)
		}
		if h.sup.sets != 0 {
			t.Fatalf("probe %d: got %d SetProxy calls, want 0", i, h.sup.sets)
		}
		h.now = h.now.Add(time.Minute)
	}

	if got := h.run(t); got.Action != "set" {
		t.Fatalf("final probe: got %q (%s), want set", got.Action, got.Reason)
	}
}

// Removing a proxy when nothing works would strand the device entirely.
func TestDynamicKeepsProxyWhenDirectPathAlsoFails(t *testing.T) {
	cfg := staticProxy()
	cfg.Mode = store.ModeDynamic
	h := newHarness(t, cfg)
	h.sup.proxy = supervisor.Proxy{Type: "http-connect", IP: "10.0.0.9", Port: 8080}.Normalized()
	h.state.Update(func(s *store.State) { s.Applied = fingerprint(h.cfg.Config().Proxy) })

	h.probe = func() proxyprobe.Result { return proxyprobe.Result{OK: false, Failed: "tcp"} }

	for i := 0; i <= store.DefaultWatchdog().RequiredFailures; i++ {
		got := h.run(t)
		if got.Action == "clear" {
			t.Fatalf("pass %d: cleared the proxy even though the direct path also fails (%s)", i, got.Reason)
		}
		h.now = h.now.Add(time.Minute)
	}
	if h.sup.clears != 0 {
		t.Fatalf("got %d ClearProxy calls, want 0", h.sup.clears)
	}
}

func TestAutoRebootOnlyWhenEnabled(t *testing.T) {
	h := newHarness(t, staticProxy())
	h.run(t)
	if h.sup.reboots != 0 {
		t.Fatalf("got %d reboots with auto-reboot off, want 0", h.sup.reboots)
	}

	h2 := newHarness(t, staticProxy())
	h2.cfg.Update(func(c *store.Config) { c.Watchdog.AutoReboot = true })
	h2.run(t)
	if h2.sup.reboots != 1 {
		t.Fatalf("got %d reboots with auto-reboot on, want 1", h2.sup.reboots)
	}
}

func TestOffModeClearsAnExistingProxy(t *testing.T) {
	h := newHarness(t, store.ProxyConfig{Mode: store.ModeOff})
	h.sup.proxy = supervisor.Proxy{Type: "socks5", IP: "10.0.0.9", Port: 1080}

	if got := h.run(t); got.Action != "clear" {
		t.Fatalf("got %q (%s), want clear", got.Action, got.Reason)
	}
	h.now = h.now.Add(time.Hour)
	if got := h.run(t); got.Action != "none" {
		t.Fatalf("second pass: got %q, want none", got.Action)
	}
}
