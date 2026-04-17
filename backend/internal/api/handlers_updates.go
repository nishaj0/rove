package api

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// returns latest release metadata for ui update checks.
func (s *Server) handleUpdatesLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.updater == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("update provider is unavailable"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()

	release, err := s.updater.LatestRelease(ctx)
	if err != nil {
		_ = s.logs.Write("error", "updates", err.Error())
		writeErr(w, http.StatusServiceUnavailable, errors.New("failed to fetch latest release"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"tagName": release.TagName,
		"htmlUrl": release.HTMLURL,
	})
}
