package proxyprobe

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// httpConnect opens a tunnel through an HTTP proxy with the CONNECT method.
// The returned reader must be used for anything read afterwards: the proxy's
// response may have been read past the header boundary into the buffer.
func httpConnect(conn net.Conn, target, user, pass string) (*bufio.Reader, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "CONNECT %s HTTP/1.1\r\n", target)
	fmt.Fprintf(&b, "Host: %s\r\n", target)
	b.WriteString("Proxy-Connection: Keep-Alive\r\n")
	b.WriteString("User-Agent: netcfgd\r\n")
	if user != "" || pass != "" {
		cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		fmt.Fprintf(&b, "Proxy-Authorization: Basic %s\r\n", cred)
	}
	b.WriteString("\r\n")

	if _, err := conn.Write([]byte(b.String())); err != nil {
		return nil, fmt.Errorf("send CONNECT: %w", err)
	}

	br := bufio.NewReader(conn)
	// http.ReadResponse needs the request method to know that a 2xx CONNECT
	// response has no body.
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		return nil, fmt.Errorf("read CONNECT reply: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		return br, nil
	case resp.StatusCode == http.StatusProxyAuthRequired:
		scheme := resp.Header.Get("Proxy-Authenticate")
		if scheme == "" {
			scheme = "unknown"
		}
		// A proxy returns 407 both when no credentials were sent and when the
		// ones sent were wrong. Only the client knows which, and the two lead
		// an operator to completely different actions, so distinguish here.
		if user == "" && pass == "" {
			return nil, fmt.Errorf("proxy requires authentication but none are configured (scheme: %s)", scheme)
		}
		return nil, fmt.Errorf("proxy rejected the credentials (scheme: %s)", scheme)
	case resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("proxy forbade this destination (403 %s)", resp.Status)
	case resp.StatusCode == http.StatusMethodNotAllowed:
		return nil, fmt.Errorf("proxy does not allow CONNECT (405); it may be a relay-only proxy, try type http-relay")
	default:
		return nil, fmt.Errorf("proxy returned %s", resp.Status)
	}
}

// isAuthStage reports whether a CONNECT failure was an authentication problem,
// so it can be attributed to the auth stage rather than to connect. Which stage
// a failure lands on is the difference between "fix the password" and "ask IT
// to allow this destination", so it is worth getting right.
func isAuthStage(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "requires authentication") ||
		strings.Contains(msg, "rejected the credentials")
}
