package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie = "netcfgd_session"
	sessionTTL    = 12 * time.Hour
)

// session is the decoded contents of a session cookie.
type session struct {
	User    string
	Nonce   string
	Expires time.Time
}

// sign returns the HMAC of a payload under the store's session key. Rotating
// that key on a password change invalidates every existing session.
func (a *App) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(a.store.SessionKey()))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *App) issueSession(w http.ResponseWriter, user string) error {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate session nonce: %w", err)
	}
	payload := fmt.Sprintf("%s|%s|%d", user,
		base64.RawURLEncoding.EncodeToString(nonce), time.Now().Add(sessionTTL).Unix())
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + a.sign(payload)

	http.SetCookie(w, &http.Cookie{
		Name:  sessionCookie,
		Value: value,
		Path:  "/",
		// The console is reached over plain HTTP on a LAN address, so Secure
		// cannot be set without locking operators out. HttpOnly and a strict
		// SameSite still remove the cookie from script access and from
		// cross-site requests.
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return nil
}

func (a *App) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: -1,
	})
}

// readSession validates the cookie and returns the session it encodes.
func (a *App) readSession(r *http.Request) (session, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return session{}, false
	}
	encoded, sig, found := strings.Cut(c.Value, ".")
	if !found {
		return session{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return session{}, false
	}
	payload := string(raw)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(a.sign(payload))) != 1 {
		return session{}, false
	}

	parts := strings.Split(payload, "|")
	if len(parts) != 3 {
		return session{}, false
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Now().After(time.Unix(exp, 0)) {
		return session{}, false
	}
	return session{User: parts[0], Nonce: parts[1], Expires: time.Unix(exp, 0)}, true
}

// csrfToken derives a per-session token. It is bound to the session nonce, so a
// token from one session cannot be replayed in another.
func (a *App) csrfToken(s session) string { return a.sign(s.Nonce + "|csrf") }

func (a *App) checkCSRF(r *http.Request, s session) bool {
	got := r.FormValue("csrf")
	if got == "" {
		got = r.Header.Get("X-CSRF-Token")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(a.csrfToken(s))) == 1
}

// loginLimiter throttles login attempts per client address. A device on a
// building LAN is reachable by anything else on that LAN, so an unthrottled
// login form is a standing invitation to guess the password.
type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func newLoginLimiter(limit int, window time.Duration) *loginLimiter {
	return &loginLimiter{attempts: map[string][]time.Time{}, limit: limit, window: window}
}

// allow reports whether another attempt may be made, and how long to wait if not.
func (l *loginLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.attempts[key] = kept

	if len(kept) >= l.limit {
		return false, kept[0].Add(l.window).Sub(now)
	}
	return true, 0
}

// record notes a failed attempt. Successful logins do not count against the
// limit, so an operator who mistypes once is not locked out after logging in.
func (l *loginLimiter) record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[key] = append(l.attempts[key], time.Now())
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// clientIP identifies the caller for rate limiting. Proxy headers are ignored
// deliberately: this console is reached directly on a LAN, so trusting
// X-Forwarded-For would let a caller trivially reset their own limit.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
