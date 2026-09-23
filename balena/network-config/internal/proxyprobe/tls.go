package proxyprobe

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Interception describes a proxy that is re-encrypting TLS.
//
// These are ordinary on corporate networks — Zscaler, Palo Alto, Fortinet and
// friends all terminate TLS and re-issue certificates from a private CA. A
// device only works behind one if it trusts that CA, so the useful thing is not
// "certificate rejected" but the identity of the CA doing it and the exact
// material needed to trust it.
type Interception struct {
	// LeafIssuer is who signed the certificate presented for the destination.
	// On a re-encrypting network this names the interceptor.
	LeafIssuer string `json:"leafIssuer"`
	// Chain is every certificate the server actually presented, leaf first.
	Chain []CertInfo `json:"chain"`
	// RootCA is the self-signed CA from the chain, when one was presented.
	//
	// Usually it is not. TLS makes sending the root optional and servers
	// normally omit it, because a client is expected to hold it already — so
	// the certificate that would need to be trusted is generally not
	// obtainable from the handshake. CAPresented says which case this is, and
	// nothing is offered as trust material unless it really is a CA.
	RootCA      *CertInfo `json:"rootCa,omitempty"`
	CAPresented bool      `json:"caPresented"`
	// Trusted reports whether the chain validated against our trust store,
	// meaning interception is configured and working rather than blocking.
	Trusted bool `json:"trusted"`
}

// CertInfo describes one presented certificate.
type CertInfo struct {
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	IsCA        bool      `json:"isCa"`
	SelfSigned  bool      `json:"selfSigned"`
	Fingerprint string    `json:"fingerprint"`
	NotAfter    time.Time `json:"notAfter"`
	// PEM and Base64 are set only for a certificate that is genuinely a
	// self-signed CA, so nothing that cannot serve as trust material is ever
	// offered as though it could.
	PEM    string `json:"pem,omitempty"`
	Base64 string `json:"base64,omitempty"`
}

var (
	trustMu    sync.RWMutex
	extraRoots *x509.CertPool
	trustNote  string
)

// LoadTrustedCAs adds PEM certificates to the pool used to verify probes.
//
// The container ships the usual public roots. A device behind an intercepting
// proxy additionally needs that proxy's CA, or every check reports a
// certificate failure and the endpoint checklist is useless exactly where it
// matters most. Note that balenaOS's own "balenaRootCA" installs into the
// *host* trust store, which containers do not share, so it has to be given
// here too.
func LoadTrustedCAs(pemBytes []byte, note string) error {
	if len(strings.TrimSpace(string(pemBytes))) == 0 {
		return nil
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pemBytes) {
		return fmt.Errorf("no usable certificates found in %s", note)
	}

	trustMu.Lock()
	defer trustMu.Unlock()
	extraRoots = pool
	trustNote = note
	return nil
}

// LoadTrustedCAsFile reads a PEM bundle from disk, ignoring a missing file.
func LoadTrustedCAsFile(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read CA bundle %s: %w", path, err)
	}
	return LoadTrustedCAs(data, path)
}

// TrustNote describes where the extra CAs came from, for the console.
func TrustNote() string {
	trustMu.RLock()
	defer trustMu.RUnlock()
	return trustNote
}

// roots returns the pool to verify against: the system roots, plus anything
// loaded through LoadTrustedCAs.
func roots() *x509.CertPool {
	trustMu.RLock()
	defer trustMu.RUnlock()
	if extraRoots != nil {
		return extraRoots.Clone()
	}
	return nil // nil means "use the system pool"
}

// verifyChain checks a handshake's certificates the way crypto/tls would, but
// against our pool. The handshake itself is done with verification deferred so
// the chain can be inspected even when it does not validate; this function is
// what actually enforces trust, and every non-reachability probe calls it.
func verifyChain(state tls.ConnectionState, host string) error {
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("the server presented no certificate")
	}
	intermediates := x509.NewCertPool()
	for _, cert := range state.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}
	_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
		DNSName:       host,
		Roots:         roots(),
		Intermediates: intermediates,
	})
	return err
}

// describeInterception summarises the presented chain so an operator can see
// who is intercepting, and obtain the CA when — unusually — it was sent.
func describeInterception(state tls.ConnectionState, trusted bool) *Interception {
	if len(state.PeerCertificates) == 0 {
		return nil
	}

	out := &Interception{
		LeafIssuer: nameOf(state.PeerCertificates[0].Issuer.CommonName,
			state.PeerCertificates[0].Issuer.Organization),
		Trusted: trusted,
	}

	for _, cert := range state.PeerCertificates {
		sum := sha256.Sum256(cert.Raw)
		selfSigned := cert.Subject.String() == cert.Issuer.String()

		info := CertInfo{
			Subject:     nameOf(cert.Subject.CommonName, cert.Subject.Organization),
			Issuer:      nameOf(cert.Issuer.CommonName, cert.Issuer.Organization),
			IsCA:        cert.IsCA,
			SelfSigned:  selfSigned,
			Fingerprint: colonHex(sum[:]),
			NotAfter:    cert.NotAfter,
		}

		// Only a self-signed CA is usable as trust material. Offering anything
		// else would produce a value that looks right and does nothing.
		if cert.IsCA && selfSigned {
			pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
			info.PEM = string(pemBytes)
			info.Base64 = base64.StdEncoding.EncodeToString(pemBytes)
			if out.RootCA == nil {
				root := info
				out.RootCA = &root
				out.CAPresented = true
			}
		}
		out.Chain = append(out.Chain, info)
	}
	return out
}

func nameOf(cn string, org []string) string {
	switch {
	case cn != "" && len(org) > 0 && org[0] != cn:
		return fmt.Sprintf("%s (%s)", cn, org[0])
	case cn != "":
		return cn
	case len(org) > 0:
		return org[0]
	}
	return "unnamed"
}

func colonHex(b []byte) string {
	s := strings.ToUpper(hex.EncodeToString(b))
	var out strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			out.WriteByte(':')
		}
		out.WriteString(s[i : i+2])
	}
	return out.String()
}

// untrustedAuthority reports whether a verification failure was about the chain
// not reaching a trusted root, as opposed to an expired or misnamed
// certificate. Only the former means "a proxy is intercepting".
func untrustedAuthority(err error) bool {
	if err == nil {
		return false
	}
	var unknown x509.UnknownAuthorityError
	if asErr(err, &unknown) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown authority") ||
		strings.Contains(msg, "certificate signed by unknown authority")
}

// publiclyRooted reports whether a chain validates against the public roots
// alone, ignoring any CA we were handed. Used to tell "ordinary TLS" apart from
// "interception that we have been configured to trust".
func publiclyRooted(state tls.ConnectionState) bool {
	if len(state.PeerCertificates) == 0 {
		return false
	}
	intermediates := x509.NewCertPool()
	for _, cert := range state.PeerCertificates[1:] {
		intermediates.AddCert(cert)
	}
	_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
		Roots:         nil, // system pool only
		Intermediates: intermediates,
		// Identity was already checked by verifyChain; this only asks which
		// pool the chain terminates in.
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	return err == nil
}
