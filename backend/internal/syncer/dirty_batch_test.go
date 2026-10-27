package syncer

import (
	"os"
	"path/filepath"
	"testing"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
)

func TestFileEventsAreBatchedUntilFlush(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg, err := config.New(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	logs, err := logstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logs.Close() })
	saves := filepath.Join(root, "saves")
	exe := filepath.Join(root, "game.exe")
	if err := os.MkdirAll(saves, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	game, err := cfg.AddGame("Batch", saves, exe, "")
	if err != nil {
		t.Fatal(err)
	}
	engine := New(cfg, logs, false)

	for _, name := range []string{"a.sav", "b.sav", "a.sav"} {
		engine.NotifyFileEvent(filepath.Join(saves, name))
	}
	if n, _ := logs.DirtyFileCount(game.ID); n != 0 {
		t.Fatalf("events must not hit SQLite before flush, got %d dirty rows", n)
	}
	engine.flushDirty()
	if n, _ := logs.DirtyFileCount(game.ID); n != 2 {
		t.Fatalf("expected 2 distinct dirty files after flush, got %d", n)
	}
}
