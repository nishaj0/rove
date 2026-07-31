package api

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) handleGoogleDriveStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.googleDrive == nil {
		writeJSON(w, http.StatusOK, map[string]any{"connected": false, "accountEmail": nil})
		return
	}
	status := s.googleDrive.Status()
	writeJSON(w, http.StatusOK, map[string]any{"connected": status.Connected, "accountEmail": nullableEmail(status.AccountEmail), "reconnectRequired": status.ReconnectRequired})
}

func (s *Server) handleGoogleDriveConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.googleDrive == nil {
		writeErr(w, http.StatusServiceUnavailable, context.Canceled)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Minute)
	defer cancel()
	status, err := s.googleDrive.Connect(ctx)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_ = s.logs.Write("info", "google_drive", "Google Drive account connected")
	writeJSON(w, http.StatusOK, map[string]any{"connected": status.Connected, "accountEmail": nullableEmail(status.AccountEmail), "reconnectRequired": false})
}

func (s *Server) handleGoogleDriveDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.googleDrive == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.googleDrive.Disconnect(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.logs.Write("info", "google_drive", "Google Drive account disconnected")
	w.WriteHeader(http.StatusNoContent)
}

func nullableEmail(value string) any {
	if value == "" {
		return nil
	}
	return value
}
