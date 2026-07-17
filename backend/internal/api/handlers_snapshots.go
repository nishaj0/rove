package api

import (
	"errors"
	"net/http"
	"strings"

	"cloud-saver/backend/internal/models"
)

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if s.snapshots == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("snapshot service is unavailable"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
		if gameID == "" {
			writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
			return
		}
		snapshots, err := s.snapshots.List(gameID)
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "game not found" {
				status = http.StatusNotFound
			}
			writeErr(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshots)
	case http.MethodPost:
		var req models.SnapshotCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		result, err := s.snapshots.CreateManual(req.GameID)
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "game not found" {
				status = http.StatusNotFound
			} else if err.Error() == "gameId is required" {
				status = http.StatusBadRequest
			}
			writeErr(w, status, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":   result.Status,
			"snapshot": result.Snapshot,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSnapshotRestore(w http.ResponseWriter, r *http.Request) {
	if s.snapshots == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("snapshot service is unavailable"))
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.SnapshotRestoreRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if s.monitor != nil && s.monitor.IsRunning(req.GameID) {
		writeErr(w, http.StatusConflict, errors.New("cannot restore while the game is running"))
		return
	}
	if err := s.snapshots.Restore(req.GameID, req.SnapshotID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "game not found" || err.Error() == "snapshot not found" {
			status = http.StatusNotFound
		} else if err.Error() == "gameId is required" || err.Error() == "snapshotId is required" {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSnapshotByID(w http.ResponseWriter, r *http.Request) {
	if s.snapshots == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("snapshot service is unavailable"))
		return
	}
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshotID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/snapshots/"))
	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if snapshotID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("snapshotId is required"))
		return
	}
	if gameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	if err := s.snapshots.Delete(gameID, snapshotID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "game not found" || err.Error() == "snapshot not found" {
			status = http.StatusNotFound
		} else if err.Error() == "gameId is required" || err.Error() == "snapshotId is required" {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
