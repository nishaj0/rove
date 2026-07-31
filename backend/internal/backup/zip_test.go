package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"cloud-saver/backend/internal/models"
)

func TestExportAndReplaceImportRoundTrip(t *testing.T) {
	root := t.TempDir()
	save := filepath.Join(root, "save")
	if err := os.MkdirAll(filepath.Join(save, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(save, "slot.dat"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(save, "nested", "config.ini"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	game := models.Game{ID: "g1", Name: `My: Game?`, Path: save}
	service := New()
	archivePath := filepath.Join(root, "exports")
	created, err := service.ExportGame(game, archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(created) != ".zip" || filepath.Base(created)[:3] != "My_" {
		t.Fatalf("unexpected export path: %s", created)
	}

	if err := os.WriteFile(filepath.Join(save, "old.dat"), []byte("remove me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.ImportGame(game, created, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(save, "old.dat")); !os.IsNotExist(err) {
		t.Fatalf("replace import left old file: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(save, "nested", "config.ini"))
	if err != nil || string(content) != "two" {
		t.Fatalf("nested file was not restored: %q, %v", content, err)
	}
}

func TestImportRejectsZipPathTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "unsafe.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../../outside.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("must not escape"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(root, "save")
	err = New().ImportGame(models.Game{Path: destination}, archivePath, true)
	if err == nil {
		t.Fatal("unsafe ZIP path was accepted")
	}
	if _, statErr := os.Stat(filepath.Join(root, "outside.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("ZIP traversal created an outside file: %v", statErr)
	}
}

func TestSanitizeFilenameUsesStableFallbacks(t *testing.T) {
	t.Parallel()
	if got := sanitizeFilename(`  <>:"/\\|?*  `); got != "game" {
		t.Fatalf("sanitizeFilename(all invalid) = %q, want game", got)
	}
	if got := sanitizeFilename("My Save  File"); got != "My_Save_File" {
		t.Fatalf("sanitizeFilename(spaces) = %q", got)
	}
	if got := autoBackupFolderName(models.Game{Name: "Game", ID: "abc"}); got != "Game__abc" {
		t.Fatalf("auto backup folder = %q", got)
	}
}
