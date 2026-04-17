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
	writeJSON(w, http.StatusOK, map[string]any{
		"debug":          s.debug,
		"pid":            os.Getpid(),
		"startedAt":      s.startedAt,
		"uptimeSec":      uptime,
		"port":           s.cfg.Get().Port,
		"configPath":     s.configPath,
		"sqlitePath":     s.sqlitePath,
		"exePath":        s.exePath,
		"startupCommand": s.startupCommand,
	})
}
