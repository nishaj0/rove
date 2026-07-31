package api

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestPrepareUpdateRequiresPOST(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/lifecycle/prepare-update", nil)
	response := httptest.NewRecorder()
	(&Server{}).handlePrepareUpdate(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestPrepareUpdateRejectsUnavailableShutdown(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/lifecycle/prepare-update", nil)
	response := httptest.NewRecorder()
	(&Server{}).handlePrepareUpdate(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestPrepareUpdateRespondsBeforeShutdownCallback(t *testing.T) {
	var called atomic.Bool
	server := &Server{prepareUpdate: func() { called.Store(true) }}
	request := httptest.NewRequest(http.MethodPost, "/lifecycle/prepare-update", nil)
	response := httptest.NewRecorder()
	server.handlePrepareUpdate(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	deadline := time.Now().Add(time.Second)
	for !called.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !called.Load() {
		t.Fatal("shutdown callback was not invoked")
	}
}
