package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sse writes a server-sent event stream. The console uses EventSource, which
// needs no library on the browser side — the whole live-update mechanism is a
// dozen lines here and about the same in app.js.
type sse struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSE(w http.ResponseWriter) (*sse, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming is not supported by this connection")
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// Nothing should buffer a stream whose whole purpose is to arrive early.
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &sse{w: w, flusher: flusher}, nil
}

func (s *sse) send(event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, payload)
	s.flusher.Flush()
}

func (s *sse) comment(text string) {
	fmt.Fprintf(s.w, ": %s\n\n", text)
	s.flusher.Flush()
}

func (s *sse) close() {}
