package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLoopbackOrigin(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"http://localhost:5173":       true,
		"https://desktop.localhost":   true,
		"http://127.0.0.1:5173":       true,
		"http://[::1]:5173":           true,
		"http://192.168.1.10:5173":    false,
		"https://localhost.evil.test": false,
		"not a url":                   false,
		"file:///tmp/index.html":      false,
	}
	for origin, want := range tests {
		if got := isLoopbackOrigin(origin); got != want {
			t.Errorf("isLoopbackOrigin(%q) = %t, want %t", origin, got, want)
		}
	}
}

func TestLocalhostOnlyRejectsNonLoopbackClients(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := localhostOnly(next)

	for _, remoteAddr := range []string{"127.0.0.1:7331", "[::1]:7331", "localhost:7331"} {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.RemoteAddr = remoteAddr
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusNoContent {
			t.Errorf("remote %q got %d, want %d", remoteAddr, response.Code, http.StatusNoContent)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.RemoteAddr = "10.0.0.5:7331"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-loopback client got %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestLoopbackCORSSetsHeadersAndRejectsForeignOrigins(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := withLoopbackCORS(next)

	request := httptest.NewRequest(http.MethodOptions, "/status", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" || response.Header().Get("Access-Control-Allow-Headers") != "Content-Type, X-Test" {
		t.Fatalf("unexpected preflight response: %d %#v", response.Code, response.Header())
	}

	foreign := httptest.NewRequest(http.MethodGet, "/status", nil)
	foreign.Header.Set("Origin", "https://evil.example")
	foreignResponse := httptest.NewRecorder()
	handler.ServeHTTP(foreignResponse, foreign)
	if foreignResponse.Code != http.StatusForbidden {
		t.Fatalf("foreign origin got %d, want %d", foreignResponse.Code, http.StatusForbidden)
	}
}
