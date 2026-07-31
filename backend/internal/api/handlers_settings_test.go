package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"cloud-saver/backend/internal/config"
)

func TestSettingsSaveIgnoresLegacyGoogleDriveStatusFields(t *testing.T) {
	t.Parallel()
	manager, err := config.New(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.SetGoogleDriveConnection("player@example.test", "protected-token"); err != nil {
		t.Fatal(err)
	}
	settings := manager.Get()
	payload, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	googleDrive := raw["googleDrive"].(map[string]any)
	googleDrive["oauthClientSource"] = "oauth_json"
	googleDrive["oauthCredentialSource"] = "oauth_json"
	payload, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{cfg: manager}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewReader(payload))
	server.handleSettings(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected legacy payload to save, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := manager.Get().GoogleDrive.EncryptedRefreshToken; got != "protected-token" {
		t.Fatal("settings save must preserve the protected refresh token")
	}
}

func TestSettingsSaveAcceptsPartialUpdate(t *testing.T) {
	t.Parallel()
	manager, err := config.New(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	deviceID := manager.Get().DeviceID
	server := &Server{cfg: manager}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/settings", bytes.NewBufferString(`{"themeMode":"dark"}`))
	server.handleSettings(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected partial payload to save, got %d: %s", recorder.Code, recorder.Body.String())
	}
	settings := manager.Get()
	if settings.ThemeMode != "dark" || settings.DeviceID != deviceID {
		t.Fatal("partial update did not preserve daemon-owned settings")
	}
}
