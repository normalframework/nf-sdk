// Package activity keeps a bounded, in-memory record of everything netcfgd
// does. The console renders it, and every entry is also written to stdout so it
// reaches `balena logs` — which is often the only view available when the
// console itself is unreachable because the network is misconfigured.
package activity

import (
	"log"
	"sync"
	"time"
)

// Level classifies an entry.
type Level string

const (
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
	// Change marks an action that altered the device's configuration. These
	// are the entries an operator scrolls back to find.
	Change Level = "change"
)

// Entry is one logged event.
type Entry struct {
	At      time.Time `json:"at"`
	Level   Level     `json:"level"`
	Source  string    `json:"source"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
	// Actor is the console user responsible, or "watchdog" for automatic
	// actions. Knowing which is which matters when explaining why a device
	// reconfigured itself at 3am.
	Actor string `json:"actor,omitempty"`
}

// Log is a fixed-size ring buffer of entries.
type Log struct {
	mu      sync.RWMutex
	entries []Entry
	limit   int
	subs    map[chan Entry]struct{}
}

// New creates a log holding at most limit entries.
func New(limit int) *Log {
	if limit < 1 {
		limit = 500
	}
	return &Log{limit: limit, subs: map[chan Entry]struct{}{}}
}

// Add records an entry.
func (l *Log) Add(level Level, source, actor, message, detail string) {
	e := Entry{At: time.Now(), Level: level, Source: source, Actor: actor, Message: message, Detail: detail}

	l.mu.Lock()
	l.entries = append(l.entries, e)
	if len(l.entries) > l.limit {
		l.entries = l.entries[len(l.entries)-l.limit:]
	}
	subs := make([]chan Entry, 0, len(l.subs))
	for c := range l.subs {
		subs = append(subs, c)
	}
	l.mu.Unlock()

	if detail != "" {
		log.Printf("[%s] %s: %s (%s)", level, source, message, detail)
	} else {
		log.Printf("[%s] %s: %s", level, source, message)
	}

	// Never block on a slow subscriber; a stalled browser must not stall the
	// watchdog.
	for _, c := range subs {
		select {
		case c <- e:
		default:
		}
	}
}

func (l *Log) Infof(source, actor, message, detail string) {
	l.Add(Info, source, actor, message, detail)
}
func (l *Log) Warnf(source, actor, message, detail string) {
	l.Add(Warn, source, actor, message, detail)
}
func (l *Log) Errorf(source, actor, message, detail string) {
	l.Add(Error, source, actor, message, detail)
}
func (l *Log) Changef(source, actor, message, detail string) {
	l.Add(Change, source, actor, message, detail)
}

// Entries returns the most recent entries, newest first.
func (l *Log) Entries(limit int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if limit <= 0 || limit > len(l.entries) {
		limit = len(l.entries)
	}
	out := make([]Entry, 0, limit)
	for i := len(l.entries) - 1; i >= len(l.entries)-limit; i-- {
		out = append(out, l.entries[i])
	}
	return out
}

// Subscribe returns a channel of new entries and a function to release it.
func (l *Log) Subscribe() (<-chan Entry, func()) {
	c := make(chan Entry, 32)
	l.mu.Lock()
	l.subs[c] = struct{}{}
	l.mu.Unlock()

	return c, func() {
		l.mu.Lock()
		if _, ok := l.subs[c]; ok {
			delete(l.subs, c)
			close(c)
		}
		l.mu.Unlock()
	}
}
