package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"cloud-saver/backend/internal/autostart"
	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/syncer"
)

type Server struct {
	cfg            *config.Manager
	engine         *syncer.Engine
	logs           *logstore.Store
	backup         *backup.Service
	onGamesChanged func([]models.Game)
	exePath        string
}

func New(cfg *config.Manager, engine *syncer.Engine, logs *logstore.Store, backupSvc *backup.Service, exePath string, onGamesChanged func([]models.Game)) *Server {
	return &Server{
		cfg:            cfg,
		engine:         engine,
		logs:           logs,
		backup:         backupSvc,
		onGamesChanged: onGamesChanged,
		exePath:        exePath,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/games", s.handleGames)
	mux.HandleFunc("/games/", s.handleGameByID)
	mux.HandleFunc("/sync", s.handleSync)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/settings", s.handleSettings)
	mux.HandleFunc("/logs/recent", s.handleLogsRecent)
	mux.HandleFunc("/backup/export", s.handleBackupExport)
	mux.HandleFunc("/backup/import", s.handleBackupImport)
	return localhostOnly(mux)
}

func localhostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// keep control endpoints local-only for mvp security.
		host := r.RemoteAddr
		if !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "[::1]") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.cfg.Get().Games)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		game, err := s.cfg.AddGame(req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		_ = s.logs.Write("info", "games", "added game: "+game.Name)
		writeJSON(w, http.StatusCreated, game)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGameByID(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	if id == "" || id == "games" {
		writeErr(w, http.StatusBadRequest, errors.New("missing game id"))
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		game, err := s.cfg.UpdateGame(id, req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		_ = s.logs.Write("info", "games", "updated game: "+game.Name)
		writeJSON(w, http.StatusOK, game)
	case http.MethodDelete:
		if err := s.cfg.DeleteGame(id); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.onGamesChanged(s.cfg.Get().Games)
		_ = s.logs.Write("info", "games", "deleted game id: "+id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.engine.Trigger("api")
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.engine.Status())
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.cfg.Get())
	case http.MethodPost:
		var req models.Settings
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := s.cfg.Update(req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := autostart.SetEnabled("CloudSaver", s.exePath, req.Autostart); err != nil {
			_ = s.logs.Write("error", "autostart", err.Error())
		}
		writeJSON(w, http.StatusOK, s.cfg.Get())
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLogsRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := s.logs.Recent(limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

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
	if err := s.backup.ExportGame(game, req.DestinationPath); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.logs.Write("info", "backup", fmt.Sprintf("exported backup for %s", game.Name))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	// reject unknown payload fields to surface client mistakes early.
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
