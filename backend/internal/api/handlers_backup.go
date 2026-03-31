package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud-saver/backend/internal/models"
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
	s.engine.Trigger("backup-import")
	_ = s.logs.Write("info", "backup", fmt.Sprintf("imported backup for %s", game.Name))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
