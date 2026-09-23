// Package store persists netcfgd's configuration and watchdog state to the
// container's data volume.
//
// Durability matters more here than it usually would: applying a proxy change
// restarts balenaEngine, which restarts this container, so any state the
// watchdog holds only in memory is lost precisely when a change is in flight.
// Everything the watchdog needs to avoid repeating itself lives on disk.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Mode selects how the proxy watchdog behaves.
type Mode string

const (
	// ModeOff keeps the host free of any proxy configuration.
	ModeOff Mode = "off"
	// ModeStatic applies the configured proxy and leaves it applied.
	ModeStatic Mode = "static"
	// ModeFallback applies the proxy at boot and clears it if the device has
	// still not reached the internet after GraceMinutes. For sites where a
	// proxy may or may not be required and nobody can tell you which.
	ModeFallback Mode = "fallback"
	// ModeDynamic starts with no proxy and only applies one after the proxy
	// has been verified from userspace. The safest mode, and the slowest.
	ModeDynamic Mode = "dynamic"
)

// Valid reports whether m is a known mode.
func (m Mode) Valid() bool {
	switch m {
	case ModeOff, ModeStatic, ModeFallback, ModeDynamic:
		return true
	}
	return false
}

// ProxyConfig is the operator's intent for the host proxy.
type ProxyConfig struct {
	Mode     Mode   `json:"mode"`
	Type     string `json:"type"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Login    string `json:"login"`
	Password string `json:"password"`
	// NoProxy lists destinations that bypass the proxy. balenaOS already
	// excludes local and reserved subnets.
	NoProxy []string `json:"noProxy"`
	// DNS configures redsocks' dnsu2t plugin: "" for off, "on" for the
	// Supervisor's default, or "ADDRESS:PORT".
	DNS string `json:"dns"`
}

// WatchdogConfig bounds how eagerly the watchdog is allowed to act. Every
// default here is chosen to make the watchdog quiet: a proxy change restarts
// every container on the device, so the cost of acting too often is high and
// the cost of acting a few minutes late is usually nil.
type WatchdogConfig struct {
	// GraceMinutes is how long fallback mode waits after boot before giving up
	// on the proxy.
	GraceMinutes int `json:"graceMinutes"`
	// RequiredPasses is how many consecutive successful probes dynamic mode
	// needs before it will apply a proxy.
	RequiredPasses int `json:"requiredPasses"`
	// RequiredFailures is how many consecutive failed probes dynamic mode
	// needs before it will remove a proxy it applied.
	RequiredFailures int `json:"requiredFailures"`
	// MinDwellMinutes is the minimum time between two proxy changes.
	MinDwellMinutes int `json:"minDwellMinutes"`
	// MaxChangesPerHour trips the flap breaker, which freezes the watchdog
	// until an operator clears it.
	MaxChangesPerHour int `json:"maxChangesPerHour"`
	// IntervalSeconds is how often the reconcile loop runs.
	IntervalSeconds int `json:"intervalSeconds"`
	// ProbeTarget is the address probed to decide whether a path works.
	ProbeTarget string `json:"probeTarget"`
	// AutoReboot reboots the device after a proxy change. The Supervisor does
	// not close existing connections when the proxy changes, so without this a
	// device that was already connected keeps its direct connections until
	// they break on their own.
	AutoReboot bool `json:"autoReboot"`
	// Force bypasses balena update locks. Off by default so the watchdog never
	// interrupts an application update.
	Force bool `json:"force"`
}

// DefaultWatchdog returns the conservative defaults.
func DefaultWatchdog() WatchdogConfig {
	return WatchdogConfig{
		GraceMinutes:      10,
		RequiredPasses:    2,
		RequiredFailures:  3,
		MinDwellMinutes:   10,
		MaxChangesPerHour: 4,
		IntervalSeconds:   60,
		ProbeTarget:       "api.balena-cloud.com:443",
		AutoReboot:        false,
		Force:             false,
	}
}

// Normalize clamps values that would make the watchdog unsafe or useless.
func (w *WatchdogConfig) Normalize() {
	d := DefaultWatchdog()
	if w.GraceMinutes < 1 {
		w.GraceMinutes = d.GraceMinutes
	}
	if w.RequiredPasses < 1 {
		w.RequiredPasses = d.RequiredPasses
	}
	if w.RequiredFailures < 1 {
		w.RequiredFailures = d.RequiredFailures
	}
	// A dwell shorter than a minute would let the watchdog restart every
	// container on the device in a tight loop.
	if w.MinDwellMinutes < 1 {
		w.MinDwellMinutes = d.MinDwellMinutes
	}
	if w.MaxChangesPerHour < 1 {
		w.MaxChangesPerHour = d.MaxChangesPerHour
	}
	if w.IntervalSeconds < 10 {
		w.IntervalSeconds = d.IntervalSeconds
	}
	if strings.TrimSpace(w.ProbeTarget) == "" {
		w.ProbeTarget = d.ProbeTarget
	}
}

// Config is everything netcfgd persists about operator intent.
type Config struct {
	Proxy    ProxyConfig    `json:"proxy"`
	Watchdog WatchdogConfig `json:"watchdog"`
	Auth     Auth           `json:"auth"`
	// UpdatedAt records the last write, shown in the console.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Store reads and writes the configuration file.
type Store struct {
	path string
	mu   sync.RWMutex
	cfg  Config
}

// Open loads the configuration from dir, creating it with defaults if absent.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory %s: %w", dir, err)
	}
	s := &Store{path: filepath.Join(dir, "config.json")}

	data, err := os.ReadFile(s.path)
	switch {
	case os.IsNotExist(err):
		s.cfg = Config{Proxy: ProxyConfig{Mode: ModeOff}, Watchdog: DefaultWatchdog()}
		if err := s.write(); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	default:
		if err := json.Unmarshal(data, &s.cfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", s.path, err)
		}
	}

	if !s.cfg.Proxy.Mode.Valid() {
		s.cfg.Proxy.Mode = ModeOff
	}
	s.cfg.Watchdog.Normalize()
	return s, nil
}

// Config returns a copy of the current configuration.
func (s *Store) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Update applies fn to the configuration and persists the result.
func (s *Store) Update(fn func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.cfg)
	s.cfg.Watchdog.Normalize()
	s.cfg.UpdatedAt = time.Now()
	return s.write()
}

// write persists the configuration. The caller holds the lock.
func (s *Store) write() error {
	return writeJSONAtomic(s.path, s.cfg, 0o600)
}

// writeJSONAtomic writes via a temporary file and a rename, so a power cut or
// an engine restart mid-write cannot leave a truncated file behind.
func writeJSONAtomic(path string, v any, mode os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}
