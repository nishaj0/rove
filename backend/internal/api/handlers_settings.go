package api

import (
	"net/http"

	"cloud-saver/backend/internal/models"
)

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings := s.cfg.Get()
		settings.GoogleDrive.EncryptedRefreshToken = ""
		settings.ServiceControl = s.engine.GlobalControl()
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPost:
		var req models.SettingsUpdate
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		// Provider status, credentials, games, and device identity come from the
		// daemon. Legacy clients may still send them, but SettingsUpdate ignores
		// those fields so version skew cannot overwrite trusted state.
		next := req.Apply(s.cfg.Get())
		if err := s.cfg.Update(next); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if s.engine != nil {
			s.engine.ConfigurationChanged()
		}

		settings := s.cfg.Get()
		settings.GoogleDrive.EncryptedRefreshToken = ""
		writeJSON(w, http.StatusOK, settings)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
