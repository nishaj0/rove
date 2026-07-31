package models

import "testing"

func TestSettingsUpdateApplyDoesNotMutateTrustedFields(t *testing.T) {
	current := Settings{
		Port: 7331, DeviceID: "device-1", CloudProvider: CloudProviderGoogleDrive,
		GoogleDrive: GoogleDriveSettings{Connected: true, EncryptedRefreshToken: "secret"},
		Games:       []Game{{ID: "g1", Name: "trusted"}}, IgnorePatterns: []string{"*.tmp"},
	}
	port := 7444
	patterns := []string{"*.log", "cache"}
	next := (SettingsUpdate{Port: &port, IgnorePatterns: &patterns}).Apply(current)
	patterns[0] = "mutated-after-apply"
	if next.Port != 7444 || next.DeviceID != "device-1" || next.CloudProvider != CloudProviderGoogleDrive || next.GoogleDrive.EncryptedRefreshToken != "secret" || len(next.Games) != 1 {
		t.Fatalf("trusted fields changed: %#v", next)
	}
	if next.IgnorePatterns[0] != "*.log" {
		t.Fatalf("ignore patterns were not copied: %#v", next.IgnorePatterns)
	}
}

func TestSettingsUpdateApplyPreservesOmittedEditableFields(t *testing.T) {
	current := Settings{Port: 7331, SyncRoot: "C:/sync", ThemeMode: ThemeDark, ProtectionTarget: ProtectionTargetFolder}
	next := (SettingsUpdate{ThemeMode: func() *ThemeMode { value := ThemeLight; return &value }()}).Apply(current)
	if next.ThemeMode != ThemeLight || next.Port != current.Port || next.SyncRoot != current.SyncRoot || next.ProtectionTarget != current.ProtectionTarget {
		t.Fatalf("partial update did not preserve fields: %#v", next)
	}
}
