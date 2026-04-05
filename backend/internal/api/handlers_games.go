package api

import (
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/syncer"
)

const maxCoverImagePayload = 2_500_000

type gameResponse struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Path            string              `json:"path"`
	ExecutablePath  string              `json:"executablePath"`
	CoverImageData  string              `json:"coverImageData,omitempty"`
	BackupPolicy    models.BackupPolicy `json:"backupPolicy"`
	PlaytimeSeconds int64               `json:"playtimeSeconds"`
	LastPlayedAt    string              `json:"lastPlayedAt,omitempty"`
	Running         bool                `json:"running"`
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.toGameResponses(s.cfg.Get().Games))
	case http.MethodPost:
		var req struct {
			Name           string               `json:"name"`
			Path           string               `json:"path"`
			ExecutablePath string               `json:"executablePath"`
			CoverImageData string               `json:"coverImageData"`
			BackupPolicy   *models.BackupPolicy `json:"backupPolicy"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		coverImageData, err := normalizeCoverImageData(req.CoverImageData)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		game, err := s.cfg.AddGameWithPolicy(req.Name, req.Path, req.ExecutablePath, coverImageData, req.BackupPolicy)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		if s.monitor != nil {
			s.monitor.ForcePoll()
		}
		_ = s.logs.Write("info", "games", "added game: "+game.Name)
		writeJSON(w, http.StatusCreated, s.toGameResponse(game, 0, time.Time{}, false))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGameByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/games/")
	if id == "" || strings.Contains(id, "/") {
		writeErr(w, http.StatusBadRequest, errors.New("missing game id"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		game, ok := s.cfg.FindGame(id)
		if !ok {
			writeErr(w, http.StatusNotFound, errors.New("game not found"))
			return
		}
		stats, err := s.logs.GameStats([]string{game.ID})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		stat := stats[game.ID]
		running := s.monitor != nil && s.monitor.IsRunning(game.ID)
		pending := int64(0)
		if s.monitor != nil {
			pending = s.monitor.PendingSeconds(game.ID)
		}
		writeJSON(w, http.StatusOK, s.toGameResponse(game, stat.PlaytimeSeconds+pending, stat.LastPlayedAt, running))
	case http.MethodPut:
		var req struct {
			Name           string               `json:"name"`
			Path           string               `json:"path"`
			ExecutablePath string               `json:"executablePath"`
			CoverImageData string               `json:"coverImageData"`
			BackupPolicy   *models.BackupPolicy `json:"backupPolicy"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		coverImageData, err := normalizeCoverImageData(req.CoverImageData)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		game, err := s.cfg.UpdateGameWithPolicy(id, req.Name, req.Path, req.ExecutablePath, coverImageData, req.BackupPolicy)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		if s.monitor != nil {
			s.monitor.ForcePoll()
		}
		_ = s.logs.Write("info", "games", "updated game: "+game.Name)
		writeJSON(w, http.StatusOK, s.toGameResponse(game, 0, time.Time{}, false))
	case http.MethodDelete:
		if err := s.cfg.DeleteGame(id); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := s.logs.DeleteGameStats(id); err != nil {
			_ = s.logs.Write("error", "games", fmt.Sprintf("delete game stats failed for %s: %v", id, err))
		}
		s.onGamesChanged(s.cfg.Get().Games)
		if s.monitor != nil {
			s.monitor.ForcePoll()
		}
		_ = s.logs.Write("info", "games", "deleted game id: "+id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGamePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GameID string `json:"gameId"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.GameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	game, ok := s.cfg.FindGame(req.GameID)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	if game.ExecutablePath == "" {
		writeErr(w, http.StatusBadRequest, errors.New("executablePath is required for this game"))
		return
	}
	if s.monitor != nil && s.monitor.IsRunning(game.ID) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_running"})
		return
	}

	cmd := exec.Command(game.ExecutablePath)
	cmd.Dir = filepath.Dir(game.ExecutablePath)
	if err := cmd.Start(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	if s.monitor != nil {
		s.monitor.ForcePoll()
	}
	_ = s.logs.Write("info", "games", "launched game: "+game.Name)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "launched"})
}

func (s *Server) handleGameSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GameID string `json:"gameId"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.GameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	game, ok := s.cfg.FindGame(req.GameID)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	s.engine.TriggerGame(req.GameID, syncer.TriggerReasonAPIGame)
	_ = s.logs.Write("info", "sync", "queued game sync for: "+game.Name)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (s *Server) handleGameCover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GameID         string `json:"gameId"`
		CoverImageData string `json:"coverImageData"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.GameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	coverImageData, err := normalizeCoverImageData(req.CoverImageData)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	game, ok := s.cfg.FindGame(req.GameID)
	if !ok {
		writeErr(w, http.StatusNotFound, errors.New("game not found"))
		return
	}
	updated, err := s.cfg.UpdateGameWithPolicy(game.ID, game.Name, game.Path, game.ExecutablePath, coverImageData, nil)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	s.onGamesChanged(s.cfg.Get().Games)
	writeJSON(w, http.StatusOK, s.toGameResponse(updated, 0, time.Time{}, false))
}

func (s *Server) handleGameBackupPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GameID       string              `json:"gameId"`
		BackupPolicy models.BackupPolicy `json:"backupPolicy"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.GameID == "" {
		writeErr(w, http.StatusBadRequest, errors.New("gameId is required"))
		return
	}
	updated, err := s.cfg.UpdateGameBackupPolicy(req.GameID, req.BackupPolicy)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	s.onGamesChanged(s.cfg.Get().Games)
	if s.monitor != nil {
		s.monitor.ForcePoll()
	}
	_ = s.logs.Write("info", "games", "updated backup policy for: "+updated.Name)
	stats, statsErr := s.logs.GameStats([]string{updated.ID})
	if statsErr != nil {
		writeJSON(w, http.StatusOK, s.toGameResponse(updated, 0, time.Time{}, false))
		return
	}
	stat := stats[updated.ID]
	running := s.monitor != nil && s.monitor.IsRunning(updated.ID)
	pending := int64(0)
	if s.monitor != nil {
		pending = s.monitor.PendingSeconds(updated.ID)
	}
	writeJSON(w, http.StatusOK, s.toGameResponse(updated, stat.PlaytimeSeconds+pending, stat.LastPlayedAt, running))
}

func (s *Server) toGameResponses(games []models.Game) []gameResponse {
	if len(games) == 0 {
		return []gameResponse{}
	}
	ids := make([]string, 0, len(games))
	for _, game := range games {
		ids = append(ids, game.ID)
	}

	statsByID, err := s.logs.GameStats(ids)
	if err != nil {
		_ = s.logs.Write("error", "games", fmt.Sprintf("read game stats failed: %v", err))
		statsByID = map[string]models.GameStats{}
	}

	result := make([]gameResponse, 0, len(games))
	for _, game := range games {
		stats := statsByID[game.ID]
		running := s.monitor != nil && s.monitor.IsRunning(game.ID)
		pending := int64(0)
		if s.monitor != nil {
			pending = s.monitor.PendingSeconds(game.ID)
		}
		result = append(result, s.toGameResponse(game, stats.PlaytimeSeconds+pending, stats.LastPlayedAt, running))
	}
	return result
}

func (s *Server) toGameResponse(game models.Game, playtimeSeconds int64, lastPlayedAt time.Time, running bool) gameResponse {
	resp := gameResponse{
		ID:              game.ID,
		Name:            game.Name,
		Path:            game.Path,
		ExecutablePath:  game.ExecutablePath,
		CoverImageData:  game.CoverImageData,
		BackupPolicy:    game.BackupPolicy,
		PlaytimeSeconds: playtimeSeconds,
		Running:         running,
	}
	if !lastPlayedAt.IsZero() {
		resp.LastPlayedAt = lastPlayedAt.UTC().Format(time.RFC3339)
	}
	return resp
}

func normalizeCoverImageData(raw string) (string, error) {
	data := strings.TrimSpace(raw)
	if data == "" {
		return "", nil
	}
	if len(data) > maxCoverImagePayload {
		return "", errors.New("cover image is too large")
	}
	if !strings.HasPrefix(data, "data:image/") || !strings.Contains(data, ";base64,") {
		return "", errors.New("cover image must be a base64 image data url")
	}
	return data, nil
}
