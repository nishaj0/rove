package api

import (
	"net/http"
	"os"
	"time"
)

func (s *Server) handleDebugInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	uptime := int64(time.Since(s.startedAt).Seconds())
	startupCommandExists := false
	if s.startupCommand != "" {
		if info, err := os.Stat(s.startupCommand); err == nil && !info.IsDir() {
			startupCommandExists = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"debug":                s.debug,
		"pid":                  os.Getpid(),
		"startedAt":            s.startedAt,
		"uptimeSec":            uptime,
		"port":                 s.cfg.Get().Port,
		"configPath":           s.configPath,
		"sqlitePath":           s.sqlitePath,
		"exePath":              s.exePath,
		"startupCommand":       s.startupCommand,
		"startupCommandExists": startupCommandExists,
	})
}
