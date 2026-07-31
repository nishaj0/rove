package api

import (
	"net/http"
	"time"
)

// handlePrepareUpdate performs the final safety gate before the desktop
// updater replaces the installed binaries. The response is flushed before the
// daemon begins its normal graceful shutdown sequence.
func (s *Server) handlePrepareUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.monitor != nil && len(s.monitor.RunningIDs()) > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{
			"status": "game_running",
			"error":  "Close tracked games before installing the update.",
		})
		return
	}
	if s.engine != nil && s.engine.Status().ActiveJobID != "" {
		writeJSON(w, http.StatusConflict, map[string]string{
			"status": "protection_active",
			"error":  "Wait for the active protection job to finish before installing the update.",
		})
		return
	}
	if s.prepareUpdate == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "shutdown_unavailable",
			"error":  "The background service cannot prepare for an update.",
		})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "ready"})
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		s.prepareUpdate()
	}()
}
