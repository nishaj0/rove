package syncer

import (
	"os"
	"path/filepath"
	"testing"

	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

func TestFolderReconcileTransfersUpdatesDeletesAndConflicts(t *testing.T) {
	root := t.TempDir()
	cloud := filepath.Join(root, "cloud")
	aPath := filepath.Join(root, "device-a")
	bPath := filepath.Join(root, "device-b")
	if err := os.MkdirAll(bPath, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(aPath, "save.dat"), "first")

	if err := os.MkdirAll(filepath.Join(root, "a-db"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "b-db"), 0o755); err != nil {
		t.Fatal(err)
	}
	aLogs, err := logstore.Open(filepath.Join(root, "a-db"))
	if err != nil {
		t.Fatal(err)
	}
	defer aLogs.Close()
	bLogs, err := logstore.Open(filepath.Join(root, "b-db"))
	if err != nil {
		t.Fatal(err)
	}
	defer bLogs.Close()
	gameA := models.Game{ID: "a", SyncID: "shared-game", Path: aPath}
	gameB := models.Game{ID: "b", SyncID: "shared-game", Path: bPath}
	settingsA := models.Settings{SyncRoot: cloud, DeviceID: "device-a", SyncRetention: 10, ReviewConflicts: true}
	settingsB := models.Settings{SyncRoot: cloud, DeviceID: "device-b", SyncRetention: 10, ReviewConflicts: true}

	if _, err := Reconcile(gameA, settingsA, aLogs); err != nil {
		t.Fatalf("initial publish: %v", err)
	}
	if _, err := Reconcile(gameB, settingsB, bLogs); err != nil {
		t.Fatalf("initial download: %v", err)
	}
	mustReadEqual(t, filepath.Join(bPath, "save.dat"), "first")

	mustWrite(t, filepath.Join(aPath, "save.dat"), "second")
	if _, err := Reconcile(gameA, settingsA, aLogs); err != nil {
		t.Fatalf("publish update: %v", err)
	}
	if _, err := Reconcile(gameB, settingsB, bLogs); err != nil {
		t.Fatalf("download update: %v", err)
	}
	mustReadEqual(t, filepath.Join(bPath, "save.dat"), "second")

	if err := os.Remove(filepath.Join(aPath, "save.dat")); err != nil {
		t.Fatal(err)
	}
	if _, err := Reconcile(gameA, settingsA, aLogs); err != nil {
		t.Fatalf("publish delete: %v", err)
	}
	if _, err := Reconcile(gameB, settingsB, bLogs); err != nil {
		t.Fatalf("download delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bPath, "save.dat")); !os.IsNotExist(err) {
		t.Fatalf("save was not deleted: %v", err)
	}

	mustWrite(t, filepath.Join(aPath, "save.dat"), "shared")
	if _, err := Reconcile(gameA, settingsA, aLogs); err != nil {
		t.Fatalf("republish: %v", err)
	}
	if _, err := Reconcile(gameB, settingsB, bLogs); err != nil {
		t.Fatalf("redownload: %v", err)
	}
	mustWrite(t, filepath.Join(aPath, "save.dat"), "a-change")
	mustWrite(t, filepath.Join(bPath, "save.dat"), "b-change")
	if _, err := Reconcile(gameA, settingsA, aLogs); err != nil {
		t.Fatalf("publish conflicting update: %v", err)
	}
	result, err := Reconcile(gameB, settingsB, bLogs)
	if err != nil {
		t.Fatalf("detect conflict: %v", err)
	}
	if len(result.ConflictIDs) != 1 {
		t.Fatalf("expected one conflict, got %#v", result.ConflictIDs)
	}
	mustReadEqual(t, filepath.Join(bPath, "save.dat"), "b-change")
	if err := ResolveConflict(gameB, settingsB, bLogs, result.ConflictIDs[0], "local"); err != nil {
		t.Fatalf("resolve local conflict: %v", err)
	}
	if _, err := Reconcile(gameA, settingsA, aLogs); err != nil {
		t.Fatalf("download resolved version: %v", err)
	}
	mustReadEqual(t, filepath.Join(aPath, "save.dat"), "b-change")
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
func mustReadEqual(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}
