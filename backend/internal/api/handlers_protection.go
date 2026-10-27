package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/syncer"
)

func (s *Server) handleProtectionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if gameID == "" {
		writeJSON(w, http.StatusOK, s.engine.Status())
		return
	}
	if _, ok := s.cfg.FindGame(gameID); !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	writeJSON(w, http.StatusOK, s.engine.ProtectionStatus(gameID))
}

func (s *Server) handleProtectionControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GameID            string            `json:"gameId"`
		ProtectionEnabled bool              `json:"protectionEnabled"`
		PauseScope        models.PauseScope `json:"pauseScope"`
		Duration          string            `json:"duration"`
		Reason            string            `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	control := models.ServiceControlState{
		ProtectionEnabled: req.ProtectionEnabled,
		PauseScope:        req.PauseScope,
		Reason:            strings.TrimSpace(req.Reason),
	}
	if control.PauseScope == "" {
		control.PauseScope = models.PauseNone
	}
	switch req.Duration {
	case "", "manual":
	case "one_hour":
		control.PauseUntil = time.Now().UTC().Add(time.Hour)
	case "until_login":
		control.PauseUntilLogin = true
	default:
		writeErr(w, http.StatusBadRequest, errors.New("invalid pause duration"))
		return
	}
	if err := s.engine.SetServiceControl(strings.TrimSpace(req.GameID), control); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.GameID == "" {
		writeJSON(w, http.StatusOK, s.engine.Status())
		return
	}
	writeJSON(w, http.StatusOK, s.engine.ProtectionStatus(req.GameID))
}

func (s *Server) handleProtectionRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.engine.RetryNow()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "retrying"})
}

func (s *Server) handleRevisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if gameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	revisions, err := s.engine.Revisions(gameID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, revisions)
}

func (s *Server) handleRevisionRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GameID     string `json:"gameId"`
		RevisionID string `json:"revisionId"`
		Confirmed  bool   `json:"confirmed"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if !req.Confirmed {
		writeErr(w, http.StatusBadRequest, errors.New("restore confirmation is required"))
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
	revision, err := s.logs.Revision(req.RevisionID)
	if err != nil || revision.GameID != game.ID {
		writeErr(w, http.StatusNotFound, errors.New("revision not found"))
		return
	}
	if err := s.engine.RestoreLocalRevision(r.Context(), game, revision.ID); err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func (s *Server) handleRecoveryPoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
		if gameID == "" {
			writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
			return
		}
		points, err := s.engine.RecoveryPoints(gameID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, points)
	case http.MethodPost:
		var req struct {
			GameID string `json:"gameId"`
			Name   string `json:"name"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if _, ok := s.cfg.FindGame(req.GameID); !ok {
			writeErr(w, http.StatusNotFound, errors.New("game not found"))
			return
		}
		jobID := s.engine.CreateRecoveryPoint(req.GameID, req.Name)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued", "jobId": jobID})
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeErr(w, http.StatusBadRequest, errors.New("recovery point id is required"))
			return
		}
		if err := s.engine.DeleteRecoveryPoint(id); err != nil {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	events, err := s.logs.Activity(500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleStorageUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if _, ok := s.cfg.FindGame(gameID); !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	usage, err := s.engine.StorageUsage(gameID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (s *Server) handleStorageSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	summary, err := s.engine.StorageSummary()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	status := s.engine.Status()
	jobs := s.engine.Jobs("")
	if len(jobs) > 25 {
		jobs = jobs[:25]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"generatedAt": time.Now().UTC(),
		"debugMode":   s.debug,
		"provider":    status.Provider,
		"service":     status.ServiceControl,
		"pending":     status.Pending,
		"conflicts":   status.ConflictCount,
		"recentJobs":  jobs,
	})
}

type recoveryLibraryEntry struct {
	ProfileID       string                 `json:"profileId"`
	GameID          string                 `json:"gameId,omitempty"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version,omitempty"`
	CoverSource     string                 `json:"coverSource,omitempty"`
	CoverAvailable  bool                   `json:"coverAvailable,omitempty"`
	PlaytimeSeconds int64                  `json:"playtimeSeconds,omitempty"`
	SyncID          string                 `json:"syncId"`
	ExecutableName  string                 `json:"executableName,omitempty"`
	PathHint        string                 `json:"pathHint,omitempty"`
	LatestProtected time.Time              `json:"latestProtected"`
	FileCount       int                    `json:"fileCount"`
	TotalBytes      int64                  `json:"totalBytes"`
	AutomaticCount  int                    `json:"automaticCount"`
	RecoveryPoints  []models.RecoveryPoint `json:"recoveryPoints"`
	RequiresUpdate  bool                   `json:"requiresUpdate,omitempty"`
}

func (s *Server) handleRecoveryLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	result := []recoveryLibraryEntry{}
	seen := map[string]struct{}{}
	for _, game := range s.cfg.Get().Games {
		revisions, _ := s.logs.Revisions(game.ID, 100)
		if len(revisions) == 0 {
			continue
		}
		points, _ := s.logs.RecoveryPoints(game.ID)
		automatic := 0
		for _, revision := range revisions {
			if revision.Kind == "automatic" || revision.Kind == "final" {
				automatic++
			}
		}
		stats, _ := s.logs.GameStats([]string{game.ID})
		result = append(result, recoveryLibraryEntry{
			ProfileID:       uuid.NewSHA1(uuid.NameSpaceURL, []byte(game.SyncID)).String(),
			GameID:          game.ID,
			Name:            game.Name,
			Version:         game.Version,
			CoverSource:     game.CoverSource,
			CoverAvailable:  game.CoverImageData != "",
			PlaytimeSeconds: stats[game.ID].PlaytimeSeconds,
			SyncID:          game.SyncID,
			ExecutableName:  baseName(game.ExecutablePath),
			PathHint:        tokenizePathHint(game.Path),
			LatestProtected: revisions[0].CreatedAt,
			FileCount:       revisions[0].FileCount,
			TotalBytes:      revisions[0].TotalBytes,
			AutomaticCount:  automatic,
			RecoveryPoints:  points,
		})
		seen[game.SyncID] = struct{}{}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if profiles, err := s.engine.CloudProfiles(ctx); err == nil {
		for _, profile := range profiles {
			if _, exists := seen[profile.SyncID]; exists {
				continue
			}
			result = append(result, recoveryLibraryEntry{
				ProfileID:       uuid.NewSHA1(uuid.NameSpaceURL, []byte(profile.SyncID)).String(),
				Name:            profile.Name,
				Version:         profile.Version,
				CoverSource:     profile.CoverSource,
				CoverAvailable:  profile.CoverAvailable,
				PlaytimeSeconds: profile.PlaytimeSeconds,
				SyncID:          profile.SyncID,
				ExecutableName:  profile.ExecutableName,
				PathHint:        profile.PathHint,
				LatestProtected: profile.LatestProtected,
				FileCount:       profile.FileCount,
				TotalBytes:      profile.TotalBytes,
				RecoveryPoints:  []models.RecoveryPoint{},
				RequiresUpdate:  syncer.ProfileNeedsNewerApp(profile),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].LatestProtected.After(result[j].LatestProtected) })
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRecoveryLibraryRestoreAndTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SyncID         string `json:"syncId"`
		Destination    string `json:"destination"`
		ExecutablePath string `json:"executablePath"`
		Confirmed      bool   `json:"confirmed"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if !req.Confirmed {
		writeErr(w, http.StatusPreconditionRequired, errors.New("recovery destination confirmation is required"))
		return
	}
	if strings.TrimSpace(req.ExecutablePath) == "" {
		writeErr(w, http.StatusBadRequest, errors.New("choose the installed game executable before restoring"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	game, err := s.engine.RestoreAndTrack(ctx, req.SyncID, req.Destination, req.ExecutablePath)
	if err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	s.onGamesChanged(s.cfg.Get().Games)
	if s.monitor != nil {
		s.monitor.ForcePoll()
	}
	writeJSON(w, http.StatusCreated, s.toGameResponse(game, 0, time.Now().UTC(), false))
}

func (s *Server) handleRecoveryLibraryRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SyncID      string `json:"syncId"`
		Destination string `json:"destination"`
		Confirmed   bool   `json:"confirmed"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if !req.Confirmed {
		writeErr(w, http.StatusPreconditionRequired, errors.New("recovery destination confirmation is required"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	if err := s.engine.RestoreCloudProfile(ctx, req.SyncID, req.Destination); err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func baseName(path string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '\\' || r == '/' })
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func tokenizePathHint(path string) string {
	normalized := strings.ReplaceAll(path, "/", "\\")
	lower := strings.ToLower(normalized)
	for _, token := range []struct {
		needle string
		token  string
	}{
		{strings.ToLower(strings.TrimSpace(os.Getenv("APPDATA"))), "%APPDATA%"},
		{strings.ToLower(strings.TrimSpace(os.Getenv("LOCALAPPDATA"))), "%LOCALAPPDATA%"},
		{strings.ToLower(strings.TrimSpace(os.Getenv("USERPROFILE"))) + "\\documents", "%USERPROFILE%\\Documents"},
	} {
		if token.needle != "" && strings.HasPrefix(lower, token.needle) {
			return token.token + normalized[len(token.needle):]
		}
	}
	return ""
}
