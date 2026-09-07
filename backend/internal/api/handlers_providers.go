package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud-saver/backend/internal/models"
)

// handleGoogleDriveFolders handles GET /providers/google-drive/folders.
// Returns the top-level folders in the connected Drive account so the user
// can pick where Rove stores its data.
func (s *Server) handleGoogleDriveFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.googleDrive == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("Google Drive is not configured"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	folders, err := s.googleDrive.ListFolders(ctx)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, folders)
}

// handleGoogleDriveFolder handles POST /providers/google-drive/folder.
// Body may contain { folderId, folderName } to persist a chosen folder, or
// { createName } to create a new Drive folder and then persist it.
func (s *Server) handleGoogleDriveFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.googleDrive == nil {
		writeErr(w, http.StatusServiceUnavailable, errors.New("Google Drive is not configured"))
		return
	}
	var req struct {
		FolderID   string `json:"folderId"`
		FolderName string `json:"folderName"`
		CreateName string `json:"createName"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.CreateName != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		folder, err := s.googleDrive.CreateFolder(ctx, strings.TrimSpace(req.CreateName))
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		req.FolderID = folder.ID
		req.FolderName = folder.Name
	}
	if strings.TrimSpace(req.FolderID) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("folderId or createName is required"))
		return
	}
	if err := s.googleDrive.SetFolder(req.FolderID, req.FolderName); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	settings := s.cfg.Get()
	settings.GoogleDrive.EncryptedRefreshToken = ""
	writeJSON(w, http.StatusOK, settings.GoogleDrive)
}

// handleSettingsExport handles GET /settings/export.
// Returns a JSON snapshot of portable app settings for the user to download
// and import on another machine.
func (s *Server) handleSettingsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	settings := s.cfg.Get()
	export := models.RoveSettings{
		SchemaVersion:   1,
		IgnorePatterns:  settings.IgnorePatterns,
		SyncIntervalSec: settings.SyncIntervalSeconds,
		ExportedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="rove-settings.json"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

// handleSettingsImport handles POST /settings/import.
// Accepts a RoveSettings JSON body and applies portable fields to the current
// configuration. Credentials, games, and device identity are never touched.
func (s *Server) handleSettingsImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var imported models.RoveSettings
	if err := decodeJSON(r, &imported); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	current := s.cfg.Get()
	if imported.IgnorePatterns != nil {
		current.IgnorePatterns = imported.IgnorePatterns
	}
	if imported.SyncIntervalSec > 0 {
		current.SyncIntervalSeconds = imported.SyncIntervalSec
	}
	if err := s.cfg.Update(current); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if s.engine != nil {
		s.engine.ConfigurationChanged()
	}
	settings := s.cfg.Get()
	settings.GoogleDrive.EncryptedRefreshToken = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "imported",
		"settings": settings,
	})
}

// handleRecoveryPointRestoreToGame handles POST /recovery-points/restore-to-game.
// Restores a named recovery point's revision directly to the game's existing
// save path, so a user doesn't need to create a new game entry.
func (s *Server) handleRecoveryPointRestoreToGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RecoveryPointID string `json:"recoveryPointId"`
		GameID          string `json:"gameId"`
		Confirmed       bool   `json:"confirmed"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if !req.Confirmed {
		writeErr(w, http.StatusBadRequest, errors.New("restore confirmation is required"))
		return
	}
	if strings.TrimSpace(req.RecoveryPointID) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("recoveryPointId is required"))
		return
	}
	game, ok := s.cfg.FindGame(req.GameID)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	if s.monitor != nil && s.monitor.IsRunning(game.ID) {
		writeErr(w, http.StatusConflict, errors.New("cannot restore while the game is running"))
		return
	}
	// Look up the recovery point to find its revision ID.
	points, err := s.logs.RecoveryPoints(game.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var revisionID string
	for _, p := range points {
		if p.ID == req.RecoveryPointID {
			revisionID = p.RevisionID
			break
		}
	}
	if revisionID == "" {
		writeErr(w, http.StatusNotFound, errors.New("recovery point not found for this game"))
		return
	}
	if err := s.engine.RestoreLocalRevision(r.Context(), game, revisionID); err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored", "revisionId": revisionID})
}

