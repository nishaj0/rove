package snapshots

import (
	"os"
	"path/filepath"
	"testing"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
)

func TestManualSnapshotListRestoreDeleteAndRetention(t *testing.T) {
	root := t.TempDir()
	savePath := filepath.Join(root, "save")
	vaultPath := filepath.Join(root, "vault")
	if err := os.MkdirAll(savePath, 0o755); err != nil {
		t.Fatal(err)
	}
	executablePath := filepath.Join(root, "game.exe")
	if err := os.WriteFile(executablePath, []byte("exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(savePath, "slot.dat"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.New(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := cfg.Get()
	settings.SnapshotVaultPath = vaultPath
	settings.SnapshotMaxPerGame = 2
	if err := cfg.Update(settings); err != nil {
		t.Fatal(err)
	}
	game, err := cfg.AddGame("Example", savePath, executablePath, "")
	if err != nil {
		t.Fatal(err)
	}
	logs, err := logstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer logs.Close()
	service := New(cfg, logs)

	first, err := service.CreateManual(game.ID)
	if err != nil || first.Status != "created" {
		t.Fatalf("first snapshot = %#v, err=%v", first, err)
	}
	if err := os.WriteFile(filepath.Join(savePath, "slot.dat"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateManual(game.ID)
	if err != nil || second.Status != "created" {
		t.Fatalf("second snapshot = %#v, err=%v", second, err)
	}
	if err := os.WriteFile(filepath.Join(savePath, "slot.dat"), []byte("three"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateManual(game.ID); err != nil {
		t.Fatal(err)
	}
	list, err := service.List(game.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("retention list = %#v, err=%v", list, err)
	}
	if err := service.Restore(game.ID, second.Snapshot.ID); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(savePath, "slot.dat"))
	if err != nil || string(content) != "two" {
		t.Fatalf("restore content = %q, err=%v", content, err)
	}
	afterRestore, err := service.List(game.ID)
	if err != nil || len(afterRestore) != 2 {
		t.Fatalf("post-restore list = %#v, err=%v", afterRestore, err)
	}
	if err := service.Delete(game.ID, afterRestore[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(game.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(game.ID, afterRestore[0].ID); err == nil {
		t.Fatal("deleting the same snapshot twice should fail")
	}
}
