package api

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleGoogleDriveStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.googleDrive == nil {
		writeJSON(w, http.StatusOK, map[string]any{"connected": false, "accountEmail": nil, "ready": false})
		return
	}
	status := s.googleDrive.Status()
	ready, configurationError := s.googleDrive.Ready()
	writeJSON(w, http.StatusOK, map[string]any{
		"connected":             status.Connected,
		"accountEmail":          nullableEmail(status.AccountEmail),
		"reconnectRequired":     status.ReconnectRequired,
		"oauthClientSource":     s.googleDrive.ClientIDSource(),
		"oauthCredentialSource": s.googleDrive.CredentialSource(),
		"ready":                 ready,
		"configurationError":    configurationError,
	})
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
		_ = s.logs.Write("error", "google_drive_oauth", "connect request failed: "+err.Error())
		if s.debug {
			log.Printf("[google-drive:connect_request_failed] %v", err)
		}
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if s.engine != nil {
		// A fresh token makes previously safe, queued local revisions eligible
		// immediately; do not wait for a Windows network-change event.
		s.engine.ConfigurationChanged()
	}
	ready, configurationError := s.googleDrive.Ready()
	_ = s.logs.Write("info", "google_drive", "Google Drive account connected using "+s.googleDrive.ClientIDSource()+" OAuth client configuration")
	writeJSON(w, http.StatusOK, map[string]any{
		"connected":             status.Connected,
		"accountEmail":          nullableEmail(status.AccountEmail),
		"reconnectRequired":     false,
		"oauthClientSource":     s.googleDrive.ClientIDSource(),
		"oauthCredentialSource": s.googleDrive.CredentialSource(),
		"ready":                 ready,
		"configurationError":    configurationError,
	})
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
	if s.engine != nil {
		s.engine.ConfigurationChanged()
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
