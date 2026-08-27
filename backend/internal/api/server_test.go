package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(`{"known":true,"typo":true}`))
	var payload struct {
		Known bool `json:"known"`
	}
	if err := decodeJSON(req, &payload); err == nil {
		t.Fatal("decodeJSON accepted an unknown field")
	}
}

func TestWriteJSONUsesJSONContentType(t *testing.T) {
	t.Parallel()
	response := httptest.NewRecorder()
	writeJSON(response, http.StatusAccepted, map[string]string{"status": "queued"})
	if response.Code != http.StatusAccepted || response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected response metadata: %d %#v", response.Code, response.Header())
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload["status"] != "queued" {
		t.Fatalf("unexpected JSON payload: %s", response.Body.String())
	}
}

func TestHandlerVersionAndMethodBoundary(t *testing.T) {
	server := (&Server{}).Handler()
	get := httptest.NewRequest(http.MethodGet, "/version", nil)
	get.RemoteAddr = "127.0.0.1:7331"
	response := httptest.NewRecorder()
	server.ServeHTTP(response, get)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /version got %d: %s", response.Code, response.Body.String())
	}

	post := httptest.NewRequest(http.MethodPost, "/version", nil)
	post.RemoteAddr = "127.0.0.1:7331"
	postResponse := httptest.NewRecorder()
	server.ServeHTTP(postResponse, post)
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /version got %d, want %d", postResponse.Code, http.StatusMethodNotAllowed)
	}
}
