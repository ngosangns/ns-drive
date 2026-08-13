package api

import "net/http"

// handleRuntime returns the current runtime projection (same payload as the
// first SSE runtime:snapshot frame).
func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	respondOK(w, s.runtimeSnapshot())
}
