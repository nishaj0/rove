package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	if s.monitor != nil && s.monitor.IsRunning(req.GameID) {
		writeErr(w, http.StatusConflict, errors.New("cannot import a save while the game is running"))
		return
	}
	if !req.Merge && !req.Confirmed {
		writeErr(w, http.StatusPreconditionRequired, errors.New("replace import requires explicit confirmation"))
		return
	}
	if !req.Merge {
		recoveryJob := s.engine.CreateRecoveryPoint(game.ID, "Before ZIP import")
		recoveryCtx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if _, err := s.engine.WaitForLocalProtection(recoveryCtx, recoveryJob); err != nil {
			writeErr(w, http.StatusConflict, fmt.Errorf("could not create the required pre-import Recovery Point: %w", err))
			return
		}
	}
	if err := s.backup.ImportGame(game, req.SourcePath, req.Merge); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.engine.TriggerGame(game.ID, syncer.TriggerReasonImport)
	_ = s.logs.WriteActivity(models.ActivityEvent{
		Category: "recovery",
		Severity: "info",
		GameID:   game.ID,
		Title:    "ZIP import completed",
		Detail:   "Imported save data is being protected as a new local revision.",
	})
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
