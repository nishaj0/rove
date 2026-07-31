package api

import (
	"errors"
	"net/http"
	"strings"

	"cloud-saver/backend/internal/syncer"
)

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	jobID := s.engine.Trigger(syncer.TriggerReasonAPI)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued", "jobId": jobID})
}

func (s *Server) handleSyncConflicts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
		if gameID == "" {
			writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
			return
		}
		conflicts, err := s.logs.SyncConflicts(gameID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, conflicts)
	case http.MethodPost:
		var req struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		conflict, err := s.logs.SyncConflict(req.ID)
		if err != nil {
			writeErr(w, http.StatusNotFound, errors.New("sync conflict not found"))
			return
		}
		game, ok := s.cfg.FindGame(conflict.GameID)
		if !ok {
			writeErr(w, http.StatusNotFound, errors.New("game not found"))
			return
		}
		if s.monitor != nil && s.monitor.IsRunning(game.ID) {
			writeErr(w, http.StatusConflict, errors.New("cannot resolve a conflict while the game is running"))
			return
		}
		if err := s.engine.ResolveConflict(game, req.ID, strings.TrimSpace(req.Action)); err != nil {
			writeErr(w, http.StatusConflict, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSyncJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.engine.Jobs(r.URL.Query().Get("gameId")))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.engine.Status())
}
