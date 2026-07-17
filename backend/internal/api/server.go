package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud-saver/backend/internal/backup"
	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/gamemon"
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/snapshots"
	"cloud-saver/backend/internal/syncer"
	"cloud-saver/backend/internal/updater"
)

type Server struct {
	cfg            *config.Manager
	engine         *syncer.Engine
	logs           *logstore.Store
	backup         *backup.Service
	monitor        *gamemon.Monitor
	snapshots      *snapshots.Service
	debug          bool
	startedAt      time.Time
	configPath     string
	sqlitePath     string
	onGamesChanged func([]models.Game)
	exePath        string
	startupCommand string
	updater        *updater.Client
}

func New(cfg *config.Manager, engine *syncer.Engine, logs *logstore.Store, backupSvc *backup.Service, monitor *gamemon.Monitor, snapshotSvc *snapshots.Service, exePath string, startupCommand string, updaterClient *updater.Client, debug bool, onGamesChanged func([]models.Game)) *Server {
	return &Server{
		cfg:            cfg,
		engine:         engine,
		logs:           logs,
		backup:         backupSvc,
		monitor:        monitor,
		snapshots:      snapshotSvc,
		debug:          debug,
		startedAt:      time.Now().UTC(),
		configPath:     cfg.Path(),
		sqlitePath:     logs.Path(),
		onGamesChanged: onGamesChanged,
		exePath:        exePath,
		startupCommand: startupCommand,
		updater:        updaterClient,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/games", s.handleGames)
	mux.HandleFunc("/games/sync", s.handleGameSync)
	mux.HandleFunc("/sync/game", s.handleGameSync)
	mux.HandleFunc("/games/play", s.handleGamePlay)
	mux.HandleFunc("/games/cover", s.handleGameCover)
	mux.HandleFunc("/games/backup-policy", s.handleGameBackupPolicy)
	mux.HandleFunc("/games/pair", s.handleGamePair)
	mux.HandleFunc("/games/", s.handleGameByID)
	mux.HandleFunc("/sync", s.handleSync)
	mux.HandleFunc("/sync/jobs", s.handleSyncJobs)
	mux.HandleFunc("/sync/conflicts", s.handleSyncConflicts)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/settings", s.handleSettings)
	mux.HandleFunc("/logs/recent", s.handleLogsRecent)
	mux.HandleFunc("/debug/info", s.handleDebugInfo)
	mux.HandleFunc("/backup/export", s.handleBackupExport)
	mux.HandleFunc("/backup/import", s.handleBackupImport)
	mux.HandleFunc("/backup/history", s.handleBackupHistory)
	mux.HandleFunc("/snapshots", s.handleSnapshots)
	mux.HandleFunc("/snapshots/restore", s.handleSnapshotRestore)
	mux.HandleFunc("/snapshots/", s.handleSnapshotByID)
	mux.HandleFunc("/updates/latest", s.handleUpdatesLatest)

	handler := localhostOnly(mux)
	handler = withLoopbackCORS(handler)
	if s.debug {
		handler = s.withDebugRequestLogs(handler)
	}
	return handler
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

func (s *Server) withDebugRequestLogs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		status := http.StatusOK
		ww := &statusWriter{ResponseWriter: w, status: &status}
		next.ServeHTTP(ww, r)
		if r.Method == http.MethodOptions {
			return
		}
		if r.URL.Path == "/status" || r.URL.Path == "/logs/recent" || r.URL.Path == "/debug/info" {
			return
		}
		duration := time.Since(start)
		_ = s.logs.Write("debug", "http", fmt.Sprintf("%s %s -> %d (%s)", r.Method, r.URL.Path, status, duration.Round(time.Millisecond)))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status *int
}

func (w *statusWriter) WriteHeader(code int) {
	*w.status = code
	w.ResponseWriter.WriteHeader(code)
}
