package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud-saver/backend/internal/models"
)

func TestLoadRepairsOnlyIncompatibleFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "settings.json")
	original := DefaultSettings()
	original.ThemeMode = models.ThemeDark
	original.GoogleDrive = models.GoogleDriveSettings{
		Connected:             true,
		AccountEmail:          "player@example.test",
		EncryptedRefreshToken: "protected-token",
	}
	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	raw["port"] = json.RawMessage(`"not-a-number"`)
	raw["fieldFromNewerClient"] = json.RawMessage(`true`)
	payload, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := manager.Get()
	if settings.Port != DefaultSettings().Port {
		t.Fatalf("expected invalid port to be repaired, got %d", settings.Port)
	}
	if settings.ThemeMode != models.ThemeDark {
		t.Fatalf("expected valid theme to survive, got %q", settings.ThemeMode)
	}
	if settings.GoogleDrive.EncryptedRefreshToken != "protected-token" {
		t.Fatal("expected protected Google token to survive unrelated field repair")
	}
	if notice := manager.RecoveryNotice(); !strings.Contains(notice, "port") {
		t.Fatalf("expected safe recovery notice to name port, got %q", notice)
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(stored) {
		t.Fatal("expected repaired settings to be valid JSON")
	}
}

func TestLoadRecoversMalformedSettingsFromBackup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	manager, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := manager.Get()
	settings.ThemeMode = models.ThemeLight
	if err := manager.Update(settings); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected settings backup: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"broken":`), 0o600); err != nil {
		t.Fatal(err)
	}

	recovered, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recovered.RecoveryNotice(), "settings.json.bak") {
		t.Fatalf("expected backup recovery notice, got %q", recovered.RecoveryNotice())
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(stored) {
		t.Fatal("expected recovered primary settings to be valid JSON")
	}
	invalid, err := filepath.Glob(filepath.Join(dir, "settings.json.invalid-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(invalid) != 1 {
		t.Fatalf("expected one preserved invalid file, got %d", len(invalid))
	}
}

func TestSettingsUpdateAppliesOnlyPresentEditableFields(t *testing.T) {
	t.Parallel()
	current := DefaultSettings()
	current.DeviceID = "stable-device"
	current.GoogleDrive = models.GoogleDriveSettings{Connected: true, EncryptedRefreshToken: "protected-token"}
	dark := models.ThemeDark
	next := (models.SettingsUpdate{ThemeMode: &dark}).Apply(current)
	if next.ThemeMode != models.ThemeDark {
		t.Fatal("expected supplied theme to change")
	}
	if next.DeviceID != "stable-device" || next.GoogleDrive.EncryptedRefreshToken != "protected-token" {
		t.Fatal("runtime-owned settings changed during partial update")
	}
	if next.Port != current.Port || next.ProtectionTarget != current.ProtectionTarget {
		t.Fatal("omitted editable settings should retain their current values")
	}
}
