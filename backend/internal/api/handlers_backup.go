package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/syncer"
)

func (s *Server) handleBackupExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req models.BackupExportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	game, ok := s.cfg.FindGame(req.GameID)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	outputPath, err := s.backup.ExportGame(game, req.DestinationPath)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "destinationPath") {
			status = http.StatusBadRequest
		}
		writeErr(w, status, err)
		return
	}
	_ = s.logs.Write("info", "backup", fmt.Sprintf("exported backup for %s", game.Name))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "path": outputPath})
}

func (s *Server) handleBackupImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req models.BackupImportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	game, ok := s.cfg.FindGame(req.GameID)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	if err := s.backup.ImportGame(game, req.SourcePath, req.Merge); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.engine.Trigger(syncer.TriggerReasonImport)
	_ = s.logs.Write("info", "backup", fmt.Sprintf("imported backup for %s", game.Name))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleBackupHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if gameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	if _, ok := s.cfg.FindGame(gameID); !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := s.logs.BackupHistory(gameID, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
