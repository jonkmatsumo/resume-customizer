package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEWriter helps write Server-Sent Events
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent sends an SSE event
func (s *SSEWriter) WriteEvent(event string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(s.w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", jsonData); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// WriteError sends an error event
func (s *SSEWriter) WriteError(message string) {
	s.WriteEvent("error", map[string]string{"error": message}) //nolint:errcheck
}

// WriteComplete sends a completion event
func (s *SSEWriter) WriteComplete(runID, status string) {
	s.WriteEvent("complete", map[string]string{ //nolint:errcheck
		"run_id": runID,
		"status": status,
	})
}
