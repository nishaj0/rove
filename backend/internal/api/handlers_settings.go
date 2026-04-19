package api

import (
	"errors"
	"net/http"

	"cloud-saver/backend/internal/autostart"
	"cloud-saver/backend/internal/branding"
	"cloud-saver/backend/internal/models"
)

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings := s.cfg.Get()
		enabled, err := autostart.IsEnabled(branding.AutostartKeyName)
		if err != nil {
			_ = s.logs.Write("error", "autostart", err.Error())
		} else {
			settings.Autostart = enabled
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPost:
		var req models.Settings
		if err := decodeJSON(r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}

		previousAutostart := s.cfg.Get().Autostart
		enabled, err := autostart.IsEnabled(branding.AutostartKeyName)
		if err != nil {
			_ = s.logs.Write("error", "autostart", err.Error())
		} else {
			previousAutostart = enabled
		}

		if req.Autostart && s.startupCommand == "" {
			err := errors.New("startup target not found")
			_ = s.logs.Write("error", "autostart", err.Error())
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := autostart.SetEnabled(branding.AutostartKeyName, s.startupCommand, req.Autostart); err != nil {
			_ = s.logs.Write("error", "autostart", err.Error())
			writeErr(w, http.StatusInternalServerError, err)
			return
		}

		if err := s.cfg.Update(req); err != nil {
			if rollbackErr := autostart.SetEnabled(branding.AutostartKeyName, s.startupCommand, previousAutostart); rollbackErr != nil {
				_ = s.logs.Write("error", "autostart", rollbackErr.Error())
			}
			writeErr(w, http.StatusBadRequest, err)
			return
		}

		settings := s.cfg.Get()
		enabled, err = autostart.IsEnabled(branding.AutostartKeyName)
		if err != nil {
			_ = s.logs.Write("error", "autostart", err.Error())
		} else {
			settings.Autostart = enabled
		}
		writeJSON(w, http.StatusOK, settings)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
