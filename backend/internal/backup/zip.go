package backup

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud-saver/backend/internal/models"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) ExportGame(game models.Game, destinationZip string) error {
	if destinationZip == "" {
		return errors.New("destinationPath is required")
	}
	if err := os.MkdirAll(filepath.Dir(destinationZip), 0o755); err != nil {
		return fmt.Errorf("create destination dir: %w", err)
	}

	f, err := os.Create(destinationZip)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	return filepath.WalkDir(game.Path, func(path string, d os.DirEntry, walkErr error) error {
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
