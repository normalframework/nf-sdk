package supervisor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// newTestClient points a Client at a test server.
func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := New()
	c.base = srv.URL
	c.apiKey = "test-key"
	return c
}

// A Supervisor restart shows up as 502 "Target connection failed" from the
// host-side proxy. It clears in a second or two and must not surface as an
// error, or every boot leaves one parked in the console.
func TestGetRetriesThroughSupervisorRestart(t *testing.T) {
	var calls int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			http.Error(w, "Target connection failed", http.StatusBadGateway)
			return
		}
		w.Write([]byte(`{"network":{"proxy":{"type":"socks5","ip":"10.0.0.9","port":"1080"}}}`))
	}))

	p, err := c.GetProxy(context.Background())
	if err != nil {
		t.Fatalf("GetProxy did not survive two 502s: %v", err)
	}
	if p.IP != "10.0.0.9" || p.Port != 1080 {
		t.Fatalf("got %+v, want the proxy from the third response", p)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("made %d requests, want 3", got)
	}
}

// A failure that persists must still be reported rather than retried forever.
func TestGetGivesUpOnPersistentFailure(t *testing.T) {
	var calls int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "still down", http.StatusBadGateway)
	}))

	if _, err := c.GetProxy(context.Background()); err == nil {
		t.Fatal("expected an error once the retries were exhausted")
	}
	if got := atomic.LoadInt32(&calls); got != transientAttempts {
		t.Fatalf("made %d attempts, want %d", got, transientAttempts)
	}
}

// A real error is not a transient one and must not be retried.
func TestGetDoesNotRetryRealErrors(t *testing.T) {
	var calls int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "nope", http.StatusUnauthorized)
	}))

	if _, err := c.GetProxy(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("made %d attempts on a 401, want 1", got)
	}
}

// A PATCH must never be retried automatically: it may already have taken
// effect, and repeating it restarts every container on the device again.
func TestPatchIsNeverRetried(t *testing.T) {
	var calls int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "Target connection failed", http.StatusBadGateway)
	}))

	err := c.SetProxy(context.Background(),
		Proxy{Type: ProxySOCKS5, IP: "10.0.0.9", Port: 1080}, false)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("PATCH was sent %d times, want exactly 1", got)
	}
}

// An update lock is reported plainly rather than as a raw status line.
func TestUpdateLockIsExplained(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusLocked)
	}))

	err := c.SetProxy(context.Background(),
		Proxy{Type: ProxySOCKS5, IP: "10.0.0.9", Port: 1080}, false)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !contains(err.Error(), "update lock") {
		t.Fatalf("got %q, want it to mention the update lock", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
