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

// Intent records a proxy change that is about to be attempted.
//
// It is written to disk *before* the Supervisor is called, because the call
// restarts balenaEngine and therefore this container. Without it, a watchdog
// that is killed mid-change comes back with no memory of having acted, decides
// the change is still needed, and applies it again — restarting every container
// on the device in a loop. Reconciling this record at startup is what makes the
// watchdog safe to interrupt.
type Intent struct {
	Action      string    `json:"action"` // "set" or "clear"
	Fingerprint string    `json:"fingerprint"`
	At          time.Time `json:"at"`
}

// FallbackOutcome latches the result of fallback mode for the current boot.
type FallbackOutcome string

const (
	FallbackPending   FallbackOutcome = ""
	FallbackConfirmed FallbackOutcome = "confirmed"
	FallbackCleared   FallbackOutcome = "cleared"
)

// State is the watchdog's memory across restarts.
type State struct {
	// BootID identifies the current boot of the host. Fallback mode is scoped
	// to a boot, and this distinguishes a real reboot from this container
	// merely restarting — which it does every time a proxy change is applied.
	BootID string `json:"bootId"`

	// Applied fingerprints the configuration last successfully applied,
	// including the password. The Supervisor never returns the password on
	// GET, so a comparison against the host alone cannot detect a password
	// change; this is how we detect one.
	Applied string `json:"applied"`

	LastChangeAt time.Time   `json:"lastChangeAt"`
	Changes      []time.Time `json:"changes"`

	Frozen       bool      `json:"frozen"`
	FreezeReason string    `json:"freezeReason,omitempty"`
	FrozenAt     time.Time `json:"frozenAt,omitempty"`

	FallbackOutcome FallbackOutcome `json:"fallbackOutcome"`
	Passes          int             `json:"passes"`
	Failures        int             `json:"failures"`

	Pending *Intent `json:"pending,omitempty"`

	LastError   string    `json:"lastError,omitempty"`
	LastRunAt   time.Time `json:"lastRunAt,omitempty"`
	LastProbeOK bool      `json:"lastProbeOk"`
}

// StateStore persists watchdog state.
type StateStore struct {
	path  string
	mu    sync.RWMutex
	state State
}

// OpenState loads watchdog state from dir.
func OpenState(dir string) (*StateStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory %s: %w", dir, err)
	}
	s := &StateStore{path: filepath.Join(dir, "watchdog-state.json")}

	data, err := os.ReadFile(s.path)
	switch {
	case os.IsNotExist(err):
		// A missing file is a first run, not an error.
	case err != nil:
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	default:
		if err := json.Unmarshal(data, &s.state); err != nil {
			// Corrupt state should not stop the device from configuring its
			// network. Start clean and say so.
			s.state = State{LastError: fmt.Sprintf("previous watchdog state was unreadable and has been reset: %v", err)}
		}
	}
	return s, nil
}

// State returns a copy of the current state.
func (s *StateStore) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Update applies fn to the state and persists it.
func (s *StateStore) Update(fn func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.state)
	return writeJSONAtomic(s.path, s.state, 0o600)
}

// RecordChange notes that a proxy change happened, for dwell and flap limits.
func (s *StateStore) RecordChange(now time.Time, fingerprint string) error {
	return s.Update(func(st *State) {
		st.LastChangeAt = now
		st.Applied = fingerprint
		st.Pending = nil
		st.Changes = append(st.Changes, now)
		st.Changes = trimChanges(st.Changes, now)
	})
}

// trimChanges drops entries older than an hour, which is the flap window.
func trimChanges(changes []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-time.Hour)
	out := changes[:0]
	for _, t := range changes {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}

// ChangesInLastHour counts recent proxy changes.
func (s *StateStore) ChangesInLastHour(now time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(trimChanges(append([]time.Time(nil), s.state.Changes...), now))
}

// BootID reads the kernel's boot identifier, which changes on every reboot and
// is stable across container restarts. Falls back to an uptime-derived value if
// the file is not readable.
func BootID() string {
	if data, err := os.ReadFile("/proc/sys/kernel/random/boot_id"); err == nil {
		return strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		// Not stable, but distinguishes boots well enough to avoid latching
		// fallback mode forever if boot_id is unavailable.
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			return "uptime-" + fields[0]
		}
	}
	return "unknown"
}

// Uptime returns the host's uptime.
//
// Fallback mode counts from boot, not from process start, because this process
// is restarted by the very engine restart that applying a proxy causes. Reading
// the host's uptime is the only measure that survives that. The container gets
// the host's /proc when it runs with network_mode host and the procfs label.
func Uptime() (time.Duration, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("read /proc/uptime: %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, fmt.Errorf("/proc/uptime is empty")
	}
	var secs float64
	if _, err := fmt.Sscanf(fields[0], "%f", &secs); err != nil {
		return 0, fmt.Errorf("parse /proc/uptime %q: %w", fields[0], err)
	}
	return time.Duration(secs * float64(time.Second)), nil
}
