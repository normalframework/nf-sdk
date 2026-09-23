// Package proxyprobe tests a proxy the way an application would use it, from
// userspace, without touching the host configuration.
//
// This matters for two reasons. It lets the console verify a proxy *before*
// committing it through the Supervisor, since committing restarts every
// container on the device. And it reports which stage failed rather than a
// single boolean, because "the proxy is broken" and "the proxy rejected your
// password" and "the proxy will not let you reach port 7844" are three very
// different conversations to have with a customer's IT department.
package proxyprobe

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Stage is one step of a proxy connection attempt.
type Stage struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Skipped  bool   `json:"skipped,omitempty"`
	Millis   int64  `json:"millis"`
	Detail   string `json:"detail,omitempty"`
	Error    string `json:"error,omitempty"`
	Guidance string `json:"guidance,omitempty"`
}

// Stage names, in the order they are attempted.
const (
	StageDNS       = "dns"       // resolve the proxy's hostname
	StageTCP       = "tcp"       // reach the proxy's port
	StageHandshake = "handshake" // speak the proxy's protocol
	StageAuth      = "auth"      // authenticate to the proxy
	StageConnect   = "connect"   // ask the proxy to reach the target
	StageTLS       = "tls"       // complete TLS with the target
	StageHTTP      = "http"      // get a response from the target
)

// Result is the outcome of one probe.
type Result struct {
	Target  string    `json:"target"`
	Via     string    `json:"via"` // "direct" or "proxy <addr>"
	OK      bool      `json:"ok"`
	Stages  []Stage   `json:"stages"`
	Millis  int64     `json:"millis"`
	Started time.Time `json:"started"`
	// Failed names the stage that stopped the probe, so callers can branch on
	// the failure kind rather than parsing an error string.
	Failed string `json:"failed,omitempty"`
	// Interception is set when a proxy is re-encrypting TLS to this target,
	// whether or not its CA is trusted.
	Interception *Interception `json:"interception,omitempty"`
}

// withInterception attaches TLS interception details to a result.
func (r Result) withInterception(i *Interception) Result {
	r.Interception = i
	return r
}

// Summary renders a one-line description for logs and the activity feed.
func (r Result) Summary() string {
	if r.OK {
		return fmt.Sprintf("%s via %s: ok in %dms", r.Target, r.Via, r.Millis)
	}
	for _, s := range r.Stages {
		if !s.OK && !s.Skipped {
			return fmt.Sprintf("%s via %s: failed at %s: %s", r.Target, r.Via, s.Name, s.Error)
		}
	}
	return fmt.Sprintf("%s via %s: failed", r.Target, r.Via)
}

// recorder accumulates stages and enforces that a probe stops at the first
// failure, with everything after it marked skipped rather than silently absent.
type recorder struct {
	stages  []Stage
	started time.Time
	failed  string
}

func newRecorder() *recorder {
	return &recorder{started: time.Now()}
}

// run times a step and records it. It returns false once anything has failed,
// so callers can stop without repeating the check.
func (r *recorder) run(name string, fn func() (string, error)) bool {
	if r.failed != "" {
		r.stages = append(r.stages, Stage{Name: name, Skipped: true})
		return false
	}
	start := time.Now()
	detail, err := fn()
	s := Stage{
		Name:   name,
		Millis: time.Since(start).Milliseconds(),
		Detail: detail,
	}
	if err != nil {
		s.Error = err.Error()
		s.Guidance = guidance(name, err)
		r.failed = name
	} else {
		s.OK = true
	}
	r.stages = append(r.stages, s)
	return err == nil
}

// pass records a stage that succeeded without a separate timed step, such as
// authentication that was proven by the CONNECT it was carried in.
func (r *recorder) pass(name, detail string) {
	r.stages = append(r.stages, Stage{Name: name, OK: true, Detail: detail})
}

// skip records a stage that does not apply to this proxy type.
func (r *recorder) skip(name, why string) {
	r.stages = append(r.stages, Stage{Name: name, Skipped: true, Detail: why})
}

// stageOrder is the canonical sequence. Every result is padded out to it, so
// the console always renders the same seven rows and an operator can see at a
// glance how far the attempt got rather than having to infer it from a list
// that stops early.
var stageOrder = []string{StageDNS, StageTCP, StageHandshake, StageAuth, StageConnect, StageTLS, StageHTTP}

// pad appends any stage that was never reached, marked as not attempted.
func (r *recorder) pad() {
	seen := make(map[string]bool, len(r.stages))
	for _, s := range r.stages {
		seen[s.Name] = true
	}
	for _, name := range stageOrder {
		if !seen[name] {
			r.stages = append(r.stages, Stage{Name: name, Skipped: true, Detail: "not attempted"})
		}
	}
	// Restore canonical order; retagging can move a stage out of sequence.
	pos := make(map[string]int, len(stageOrder))
	for i, name := range stageOrder {
		pos[name] = i
	}
	sort.SliceStable(r.stages, func(i, j int) bool { return pos[r.stages[i].Name] < pos[r.stages[j].Name] })
}

func (r *recorder) result(target, via string) Result {
	r.pad()
	return Result{
		Target:  target,
		Via:     via,
		OK:      r.failed == "",
		Stages:  r.stages,
		Millis:  time.Since(r.started).Milliseconds(),
		Started: r.started,
		Failed:  r.failed,
	}
}

// guidance turns a failure into the next thing to try. The console shows this
// verbatim, so keep it to one actionable sentence.
func guidance(stage string, err error) string {
	msg := strings.ToLower(err.Error())
	switch stage {
	case StageDNS:
		return "The proxy hostname did not resolve. Use its IP address instead, or check DNS."
	case StageTCP:
		switch {
		case strings.Contains(msg, "refused"):
			return "Nothing is listening on that port. Check the address and port."
		case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
			return "Timed out rather than refused, which usually means a firewall is dropping the traffic."
		case strings.Contains(msg, "no route"):
			return "No route to the proxy. Check this interface's gateway and subnet."
		}
		return "Could not open a TCP connection to the proxy."
	case StageHandshake:
		return "The proxy did not speak the expected protocol. Check the type selected here matches."
	case StageAuth:
		if strings.Contains(msg, "none are configured") || strings.Contains(msg, "no credentials") {
			return "This proxy needs a username and password, and none are configured."
		}
		return "Check the username and password."
	case StageConnect:
		if strings.Contains(msg, "not allowed") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "403") {
			return "Blocked by policy. Ask for this host and port to be allowed."
		}
		return "The proxy connected but could not reach the destination."
	case StageTLS:
		if strings.Contains(msg, "intercepted") {
			return "This network re-encrypts TLS. Until its CA is trusted, every TLS connection from this device fails. The CA has to be trusted in two places: balenaRootCA in config.json, and this console's CA bundle."
		}
		if strings.Contains(msg, "expired") {
			return "The certificate has expired. Check this device's clock before suspecting the network."
		}
		if strings.Contains(msg, "certificate") || strings.Contains(msg, "x509") {
			return "Certificate rejected by the destination."
		}
		return "TLS did not complete with the destination."
	case StageHTTP:
		return "Connected, but the destination returned nothing usable."
	}
	return ""
}
