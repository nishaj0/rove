package gameicon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractDataURLRejectsMissingAndNonExecutableFiles(t *testing.T) {
	if _, err := ExtractDataURL(filepath.Join(t.TempDir(), "missing.exe")); err == nil {
		t.Fatal("missing executable unexpectedly produced an icon")
	}

	path := filepath.Join(t.TempDir(), "not-an-executable.exe")
	if err := os.WriteFile(path, []byte("not a PE file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractDataURL(path); err == nil {
		t.Fatal("invalid executable unexpectedly produced an icon")
	}
}
