package backup

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud-saver/backend/internal/models"
)

type Service struct{}

var invalidFilenameChars = regexp.MustCompile(`[<>:"/\\|?*]+`)

func New() *Service {
	return &Service{}
}

func (s *Service) ExportGame(game models.Game, destinationPath string) (string, error) {
	resolvedZipPath, err := resolveExportDestinationPath(game.Name, destinationPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedZipPath), 0o755); err != nil {
		return "", fmt.Errorf("create destination dir: %w", err)
	}

	f, err := os.Create(resolvedZipPath)
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	err = filepath.WalkDir(game.Path, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(game.Path, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		wf, err := zw.Create(rel)
		if err != nil {
			return err
		}
		sf, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sf.Close()

		if _, err := io.Copy(wf, sf); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return resolvedZipPath, nil
}

func resolveExportDestinationPath(gameName string, destinationPath string) (string, error) {
	trimmed := strings.TrimSpace(destinationPath)
	if trimmed == "" {
		return "", errors.New("destinationPath is required")
	}

	cleaned := filepath.Clean(trimmed)
	if strings.EqualFold(filepath.Ext(cleaned), ".zip") {
		return cleaned, nil
	}

	if info, err := os.Stat(cleaned); err == nil && info.IsDir() {
		return filepath.Join(cleaned, generatedExportFilename(gameName)), nil
	} else if err == nil && !info.IsDir() {
		return "", errors.New("destinationPath must be a .zip file path or folder path")
	}

	if strings.HasSuffix(trimmed, "\\") || strings.HasSuffix(trimmed, "/") || filepath.Ext(cleaned) == "" {
		return filepath.Join(cleaned, generatedExportFilename(gameName)), nil
	}

	return "", errors.New("destinationPath must be a .zip file path or folder path")
}

func generatedExportFilename(gameName string) string {
	safeGameName := sanitizeFilename(gameName)
	timestamp := time.Now().UTC().Format("20060102_150405")
	return fmt.Sprintf("%s_%s.zip", safeGameName, timestamp)
}

func sanitizeFilename(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "game"
	}
	withoutInvalid := invalidFilenameChars.ReplaceAllString(trimmed, "_")
	compactSpaces := strings.Join(strings.Fields(withoutInvalid), "_")
	compact := strings.Trim(compactSpaces, "._ ")
	if compact == "" {
		return "game"
	}
	return compact
}

func (s *Service) ImportGame(game models.Game, sourceZip string, merge bool) error {
	r, err := zip.OpenReader(sourceZip)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	if !merge {
		if err := os.RemoveAll(game.Path); err != nil {
			return fmt.Errorf("clear target folder: %w", err)
		}
	}
	if err := os.MkdirAll(game.Path, 0o755); err != nil {
		return fmt.Errorf("create target folder: %w", err)
	}

	root := filepath.Clean(game.Path) + string(filepath.Separator)
	for _, f := range r.File {
		target := filepath.Join(game.Path, f.Name)
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget+string(filepath.Separator), root) && cleanTarget != filepath.Clean(game.Path) {
			return fmt.Errorf("zip contains invalid path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(cleanTarget)
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rcErr := rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if rcErr != nil {
			return rcErr
		}
	}
	return nil
}
