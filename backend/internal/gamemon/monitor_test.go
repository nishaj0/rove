package gamemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/logstore"
)

func TestPollOnceFlushesTheFinalObservedPlaytimeOnExit(t *testing.T) {
	root := t.TempDir()
	cfg, err := config.New(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "game.exe")
	if err := os.WriteFile(exe, []byte("test executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	savePath := filepath.Join(root, "saves")
	if err := os.MkdirAll(savePath, 0o755); err != nil {
		t.Fatal(err)
	}
	game, err := cfg.AddGame("Crash Test", savePath, exe, "")
	if err != nil {
		t.Fatal(err)
	}
	logs, err := logstore.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logs.Close() })

	running := true
	monitor := New(cfg, logs, nil, nil)
	monitor.snapshotFn = func() map[string]bool {
		if !running {
			return map[string]bool{}
		}
		return map[string]bool{normalizePath(exe): true}
	}

	started := time.Now().Add(-10 * time.Second)
	monitor.pollOnce(started)
	monitor.pollOnce(started.Add(2 * time.Second))
	running = false
	monitor.pollOnce(started.Add(4 * time.Second))

	if monitor.IsRunning(game.ID) {
		t.Fatal("game remained marked running after its process disappeared")
	}
	stats, err := logs.GameStats([]string{game.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got := stats[game.ID].PlaytimeSeconds; got != 4 {
		t.Fatalf("expected final interval to be persisted, got %d seconds", got)
	}
}
