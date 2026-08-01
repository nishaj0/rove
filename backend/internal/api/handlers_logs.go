package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

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

func (s *Server) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	entries, err := s.logs.Recent(500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	uptime := int64(time.Since(s.startedAt).Seconds())
	debugInfo := map[string]any{
		"debug":     s.debug,
		"pid":       0, // populated by daemon on start; not available here
		"startedAt": s.startedAt,
		"uptimeSec": uptime,
		"port":      s.cfg.Get().Port,
		"configPath": s.configPath,
		"sqlitePath": s.sqlitePath,
	}

	filename := fmt.Sprintf("cloud-saver-logs-%s.zip", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	// logs.json
	if lf, err := zw.Create("logs.json"); err == nil {
		_ = json.NewEncoder(lf).Encode(entries)
	}

	// debug-info.json
	if df, err := zw.Create("debug-info.json"); err == nil {
		_ = json.NewEncoder(df).Encode(debugInfo)
	}
}
