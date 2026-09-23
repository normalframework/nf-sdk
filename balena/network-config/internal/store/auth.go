package store

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// DefaultUsername and DefaultPassword are the credentials a freshly deployed
// device starts with. They are deliberately well known, because an installer
// standing in a plant room needs to get in; MustChange forces a replacement
// before anything else can be done.
const (
	DefaultUsername = "admin"
	DefaultPassword = "normal"
)

// Auth holds the console credentials.
type Auth struct {
	Username string `json:"username"`
	// Hash is a bcrypt hash. The password itself is never stored.
	Hash []byte `json:"hash"`
	// MustChange is set until the default password has been replaced. The
	// console refuses to do anything else while it is true.
	MustChange bool `json:"mustChange"`
	// SessionKey signs session cookies. Rotating it logs everyone out, which
	// is what a password change should do.
	SessionKey string    `json:"sessionKey"`
	ChangedAt  time.Time `json:"changedAt"`
}

// EnsureAuth initialises credentials on first run.
func (s *Store) EnsureAuth() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	if len(s.cfg.Auth.Hash) == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash default password: %w", err)
		}
		s.cfg.Auth.Username = DefaultUsername
		s.cfg.Auth.Hash = hash
		s.cfg.Auth.MustChange = true
		changed = true
	}
	if s.cfg.Auth.SessionKey == "" {
		key, err := randomKey()
		if err != nil {
			return err
		}
		s.cfg.Auth.SessionKey = key
		changed = true
	}
	if changed {
		return s.write()
	}
	return nil
}

// Verify checks a username and password.
//
// The username is compared in constant time and the bcrypt comparison always
// runs, even for an unknown user, so that response timing does not reveal
// whether the username exists.
func (s *Store) Verify(username, password string) bool {
	s.mu.RLock()
	auth := s.cfg.Auth
	s.mu.RUnlock()

	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(auth.Username)) == 1
	hash := auth.Hash
	if len(hash) == 0 {
		// Compare against a fixed hash so the timing profile is unchanged.
		hash = []byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalidinv")
	}
	passOK := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	return userOK && passOK
}

// MinPasswordLength is the shortest password the console accepts. This is a
// LAN-facing device in a building, not an internet service, so the bar is
// "not guessable by someone who read the manual" rather than a policy regime.
const MinPasswordLength = 8

// SetPassword replaces the console password, clearing the must-change flag and
// rotating the session key so existing sessions are invalidated.
func (s *Store) SetPassword(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username is required")
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if password == DefaultPassword {
		return errors.New("choose a password other than the default")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	key, err := randomKey()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Auth.Username = username
	s.cfg.Auth.Hash = hash
	s.cfg.Auth.MustChange = false
	s.cfg.Auth.SessionKey = key
	s.cfg.Auth.ChangedAt = time.Now()
	return s.write()
}

// Auth returns the current credential metadata. The hash is not included.
func (s *Store) Auth() Auth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.cfg.Auth
	a.Hash = nil
	return a
}

// SessionKey returns the key used to sign session cookies.
func (s *Store) SessionKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Auth.SessionKey
}

func randomKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
