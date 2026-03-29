package api

import (
	"net/http"

	"cloud-saver/backend/internal/autostart"
	"cloud-saver/backend/internal/models"
)

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
